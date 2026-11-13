package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"collector/libs/platform/token"
)

const (
	emetteurAttendu  = "https://auth.collector.test/realms/collector"
	audienceAttendue = "interactions"
	clientAutorise   = "collector-web"
	roleVendeur      = "vendeur"
	roleAcheteur     = "acheteur"
	roleAdmin        = "admin"
)

type cleDeSignature struct {
	privee *rsa.PrivateKey
	kid    string
}

type modification func(entete, charge map[string]any)

type fabriqueDeJetons struct {
	t      testing.TB
	cle    cleDeSignature
	intrus cleDeSignature
	jwks   *httptest.Server
}

func nouvelleCle(t testing.TB, kid string) cleDeSignature {
	t.Helper()
	privee, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("génération de la clé : %v", err)
	}
	return cleDeSignature{privee: privee, kid: kid}
}

func encoder(octets []byte) string {
	return base64.RawURLEncoding.EncodeToString(octets)
}

func (c cleDeSignature) jwk() map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": c.kid,
		"n":   encoder(c.privee.N.Bytes()),
		"e":   encoder(big.NewInt(int64(c.privee.E)).Bytes()),
	}
}

func nouvelleFabriqueDeJetons(t testing.TB) *fabriqueDeJetons {
	t.Helper()
	cle := nouvelleCle(t, "cle-collector")
	contenu, err := json.Marshal(map[string]any{"keys": []map[string]string{cle.jwk()}})
	if err != nil {
		t.Fatalf("encodage du JWKS : %v", err)
	}
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(contenu)
	}))
	t.Cleanup(serveur.Close)
	return &fabriqueDeJetons{t: t, cle: cle, intrus: nouvelleCle(t, cle.kid), jwks: serveur}
}

func (f *fabriqueDeJetons) verificateur() *token.Verifier {
	f.t.Helper()
	verificateur, err := token.NewVerifier(f.t.Context(), token.Config{
		Issuer:          emetteurAttendu,
		Audience:        audienceAttendue,
		AuthorizedParty: clientAutorise,
		KeySetURL:       f.jwks.URL,
	})
	if err != nil {
		f.t.Fatalf("vérificateur de jetons : %v", err)
	}
	return verificateur
}

func (f *fabriqueDeJetons) encoderJSON(valeur any) string {
	f.t.Helper()
	octets, err := json.Marshal(valeur)
	if err != nil {
		f.t.Fatalf("encodage JSON : %v", err)
	}
	return encoder(octets)
}

func (f *fabriqueDeJetons) signer(entete, charge map[string]any, cle cleDeSignature) string {
	f.t.Helper()
	signe := f.encoderJSON(entete) + "." + f.encoderJSON(charge)
	empreinte := sha256.Sum256([]byte(signe))
	signature, err := rsa.SignPKCS1v15(rand.Reader, cle.privee, crypto.SHA256, empreinte[:])
	if err != nil {
		f.t.Fatalf("signature : %v", err)
	}
	return signe + "." + encoder(signature)
}

func revendicationsValides(sujet string, roles []string) map[string]any {
	maintenant := time.Now()
	return map[string]any{
		"iss":          emetteurAttendu,
		"aud":          []string{audienceAttendue, "account"},
		"azp":          clientAutorise,
		"typ":          "Bearer",
		"sub":          sujet,
		"iat":          maintenant.Unix(),
		"exp":          maintenant.Add(5 * time.Minute).Unix(),
		"realm_access": map[string]any{"roles": roles},
	}
}

func (f *fabriqueDeJetons) emettreAvec(cle cleDeSignature, sujet string, roles []string, modifications ...modification) string {
	f.t.Helper()
	entete := map[string]any{"alg": "RS256", "typ": "JWT", "kid": f.cle.kid}
	charge := revendicationsValides(sujet, roles)
	for _, m := range modifications {
		m(entete, charge)
	}
	return "Bearer " + f.signer(entete, charge, cle)
}

func (f *fabriqueDeJetons) emettre(sujet string, roles []string, modifications ...modification) string {
	f.t.Helper()
	return f.emettreAvec(f.cle, sujet, roles, modifications...)
}

func (f *fabriqueDeJetons) vendeur(sujet string) string {
	f.t.Helper()
	return f.emettre(sujet, []string{roleVendeur})
}

func avecRevendication(nom string, valeur any) modification {
	return func(_, charge map[string]any) {
		charge[nom] = valeur
	}
}

func sansRevendication(nom string) modification {
	return func(_, charge map[string]any) {
		delete(charge, nom)
	}
}

func avecEntete(nom string, valeur any) modification {
	return func(entete, _ map[string]any) {
		entete[nom] = valeur
	}
}

func (f *fabriqueDeJetons) jetonsInvalides(sujet, autreSujet string) map[string]string {
	f.t.Helper()
	heurePassee := time.Now().Add(-time.Hour).Unix()
	valide := f.vendeur(sujet)
	return map[string]string{
		"sans jeton":                 "",
		"schéma Basic":               "Basic dmVuZGV1cjpzZWNyZXQ=",
		"jeton illisible":            "Bearer pas.un.jeton",
		"Bearer sans jeton":          "Bearer ",
		"signé par une clé inconnue": f.emettreAvec(f.intrus, sujet, []string{roleVendeur}),
		"expiré":                     f.emettre(sujet, []string{roleVendeur}, avecRevendication("exp", heurePassee)),
		"audience de la place":       f.emettre(sujet, []string{roleVendeur}, avecRevendication("aud", []string{"place-de-marche"})),
		"émetteur étranger":          f.emettre(sujet, []string{roleVendeur}, avecRevendication("iss", "https://intrus.test/realms/collector")),
		"client non autorisé":        f.emettre(sujet, []string{roleVendeur}, avecRevendication("azp", "outil-intrus")),
		"sans expiration":            f.emettre(sujet, []string{roleVendeur}, sansRevendication("exp")),
		"algorithme none":            f.emettre(sujet, []string{roleVendeur}, avecEntete("alg", "none")),
		"sujet falsifié":             f.sujetFalsifie(autreSujet, sujet),
		"signature tronquée":         valide[:len(valide)-8],
	}
}

func (f *fabriqueDeJetons) sujetFalsifie(sujet, autreSujet string) string {
	f.t.Helper()
	parties := strings.Split(strings.TrimPrefix(f.vendeur(sujet), "Bearer "), ".")
	parties[1] = f.encoderJSON(revendicationsValides(autreSujet, []string{roleVendeur}))
	return "Bearer " + strings.Join(parties, ".")
}
