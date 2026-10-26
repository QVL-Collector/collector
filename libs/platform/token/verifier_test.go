package token_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"collector/libs/platform/token"
)

const (
	emetteur = "https://auth.collector.test/realms/collector"
	audience = "place-de-marche"
	client   = "collector-web"
	sujet    = "0b9f3c3e-6a51-4c1e-9d7a-2f4d8b1e5a77"
)

type banc struct {
	t            *testing.T
	cle          *rsa.PrivateKey
	intrus       *rsa.PrivateKey
	verificateur *token.Verifier
}

func genererCle(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	cle, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("clé : %v", err)
	}
	return cle
}

func b64(octets []byte) string {
	return base64.RawURLEncoding.EncodeToString(octets)
}

func nouveauBanc(t *testing.T) *banc {
	t.Helper()
	b := &banc{t: t, cle: genererCle(t), intrus: genererCle(t)}
	jwks, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "k1",
		"n": b64(b.cle.N.Bytes()), "e": b64(big.NewInt(int64(b.cle.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatalf("JWKS : %v", err)
	}
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(serveur.Close)
	b.verificateur, err = token.NewVerifier(t.Context(), token.Config{Issuer: emetteur, Audience: audience, AuthorizedParty: client, KeySetURL: serveur.URL})
	if err != nil {
		t.Fatalf("vérificateur : %v", err)
	}
	return b
}

func charge(roles ...string) map[string]any {
	return map[string]any{
		"iss": emetteur, "aud": audience, "azp": client, "typ": "Bearer", "sub": sujet,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
		"realm_access": map[string]any{"roles": roles},
	}
}

func (b *banc) signer(entete, revendications map[string]any, cle *rsa.PrivateKey) string {
	b.t.Helper()
	e, err := json.Marshal(entete)
	if err != nil {
		b.t.Fatal(err)
	}
	c, err := json.Marshal(revendications)
	if err != nil {
		b.t.Fatal(err)
	}
	signe := b64(e) + "." + b64(c)
	empreinte := sha256.Sum256([]byte(signe))
	signature, err := rsa.SignPKCS1v15(rand.Reader, cle, crypto.SHA256, empreinte[:])
	if err != nil {
		b.t.Fatal(err)
	}
	return signe + "." + b64(signature)
}

func (b *banc) jeton(modifier func(entete, revendications map[string]any)) string {
	entete := map[string]any{"alg": "RS256", "typ": "JWT", "kid": "k1"}
	revendications := charge("vendeur", "acheteur")
	modifier(entete, revendications)
	return b.signer(entete, revendications, b.cle)
}

func remplacer(nom string, valeur any) func(entete, revendications map[string]any) {
	return func(_, revendications map[string]any) { revendications[nom] = valeur }
}

func TestJetonValideDonneSujetEtRoles(t *testing.T) {
	b := nouveauBanc(t)
	revendications, err := b.verificateur.Verify(t.Context(), b.jeton(func(_, _ map[string]any) {}))
	if err != nil {
		t.Fatalf("jeton valide refusé : %v", err)
	}
	if revendications.Subject != sujet || !slices.Contains(revendications.Roles, "vendeur") || !revendications.HasRole("acheteur") {
		t.Fatalf("revendications inattendues : %+v", revendications)
	}
}

func TestJetonsRefuses(t *testing.T) {
	b := nouveauBanc(t)
	cas := map[string]string{
		"expiré":                  b.jeton(remplacer("exp", time.Now().Add(-time.Hour).Unix())),
		"autre émetteur":          b.jeton(remplacer("iss", "https://ailleurs.test/realms/collector")),
		"autre audience":          b.jeton(remplacer("aud", "transactions")),
		"autre client":            b.jeton(remplacer("azp", "transactions")),
		"type Refresh":            b.jeton(remplacer("typ", "Refresh")),
		"clé inconnue":            b.jeton(func(entete, _ map[string]any) { entete["kid"] = "k2" }),
		"signé par une autre clé": b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, charge("vendeur"), b.intrus),
		"algorithme none":         b64([]byte(`{"alg":"none"}`)) + "." + b64([]byte(`{"sub":"x"}`)) + ".",
		"mal formé":               "pas-un-jeton",
		"vide":                    "",
	}
	for nom, brut := range cas {
		t.Run(nom, func(t *testing.T) {
			if _, err := b.verificateur.Verify(t.Context(), brut); !errors.Is(err, token.ErrInvalidToken) {
				t.Fatalf("ErrInvalidToken attendu, obtenu %v", err)
			}
		})
	}
}

func appeler(t *testing.T, gestionnaire http.Handler, autorisation string) int {
	t.Helper()
	requete := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protege", nil)
	if autorisation != "" {
		requete.Header.Set("Authorization", autorisation)
	}
	enregistreur := httptest.NewRecorder()
	gestionnaire.ServeHTTP(enregistreur, requete)
	return enregistreur.Code
}

func TestAuthentificationEtRoleExige(t *testing.T) {
	b := nouveauBanc(t)
	atteint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := token.ClaimsFrom(r.Context()); !ok {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	journal := slog.New(slog.NewTextHandler(io.Discard, nil))
	protege := b.verificateur.Authenticate(journal, token.RequireRole("vendeur", atteint))
	acheteur := b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, charge("acheteur"), b.cle)
	cas := map[string]struct {
		autorisation string
		attendu      int
	}{
		"vendeur":      {"Bearer " + b.jeton(func(_, _ map[string]any) {}), http.StatusNoContent},
		"acheteur":     {"Bearer " + acheteur, http.StatusForbidden},
		"sans jeton":   {"", http.StatusUnauthorized},
		"schéma Basic": {"Basic YTpi", http.StatusUnauthorized},
		"jeton refusé": {"Bearer pas-un-jeton", http.StatusUnauthorized},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			if obtenu := appeler(t, protege, c.autorisation); obtenu != c.attendu {
				t.Fatalf("statut %d attendu, %d obtenu", c.attendu, obtenu)
			}
		})
	}
}
