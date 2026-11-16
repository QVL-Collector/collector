package token_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"collector/libs/platform/token"
)

const (
	clientDeService = "transactions"
	porteeReserver  = "reserver"
	sujetDuService  = "5d2c1f0e-3b4a-4c5d-8e9f-0a1b2c3d4e5f"
	roleAcheteur    = "acheteur"
)

func (b *banc) verificateurDeService(t *testing.T) *token.Verifier {
	t.Helper()
	service, err := b.verificateur.ForService(clientDeService, porteeReserver)
	if err != nil {
		t.Fatalf("vérificateur de service : %v", err)
	}
	return service
}

func (b *banc) jetonDeService(modifier func(entete, revendications map[string]any)) string {
	return b.jeton(func(entete, revendications map[string]any) {
		revendications["azp"] = clientDeService
		revendications["sub"] = sujetDuService
		revendications["scope"] = "basic " + porteeReserver
		delete(revendications, "realm_access")
		modifier(entete, revendications)
	})
}

func sansModification(_, _ map[string]any) {}

func retirer(nom string) func(entete, revendications map[string]any) {
	return func(_, revendications map[string]any) { delete(revendications, nom) }
}

func TestForServiceRefuseUneConfigurationIncomplete(t *testing.T) {
	b := nouveauBanc(t)
	cas := map[string][2]string{
		"portée vide": {clientDeService, ""},
		"client vide": {"", porteeReserver},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			if verificateur, err := b.verificateur.ForService(c[0], c[1]); err == nil || verificateur != nil {
				t.Fatalf("configuration (%q, %q) acceptée", c[0], c[1])
			}
		})
	}
}

func TestJetonDeServiceAccepteAvecLaPortee(t *testing.T) {
	b := nouveauBanc(t)
	service := b.verificateurDeService(t)
	cas := map[string]string{
		"portée parmi d'autres": "basic reserver profile",
		"portée seule":          porteeReserver,
		"espaces multiples":     "  basic   reserver ",
	}
	for nom, portee := range cas {
		t.Run(nom, func(t *testing.T) {
			revendications, err := service.Verify(t.Context(), b.jetonDeService(remplacer("scope", portee)))
			if err != nil {
				t.Fatalf("jeton de service refusé : %v", err)
			}
			if revendications.Subject != sujetDuService || len(revendications.Roles) != 0 {
				t.Fatalf("revendications inattendues : %+v", revendications)
			}
		})
	}
}

func TestJetonDeServiceRefuse(t *testing.T) {
	b := nouveauBanc(t)
	service := b.verificateurDeService(t)
	cas := map[string]string{
		"sans portée":                   b.jetonDeService(retirer("scope")),
		"portée vide":                   b.jetonDeService(remplacer("scope", "")),
		"portée voisine":                b.jetonDeService(remplacer("scope", "basic reserverx")),
		"portée préfixe":                b.jetonDeService(remplacer("scope", "reserv")),
		"portée en casse différente":    b.jetonDeService(remplacer("scope", "RESERVER")),
		"portée non textuelle (liste)":  b.jetonDeService(remplacer("scope", []string{porteeReserver})),
		"portée non textuelle (nombre)": b.jetonDeService(remplacer("scope", 1)),
		"autre client":                  b.jetonDeService(remplacer("azp", "intrus")),
		"client du site":                b.jetonDeService(remplacer("azp", client)),
		"sans azp":                      b.jetonDeService(retirer("azp")),
		"autre audience":                b.jetonDeService(remplacer("aud", "interactions")),
		"expiré":                        b.jetonDeService(remplacer("exp", int64(1))),
		"autre émetteur":                b.jetonDeService(remplacer("iss", "https://ailleurs.test/realms/collector")),
		"jeton d'utilisateur":           b.jeton(sansModification),
		"signé par une autre clé":       b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, map[string]any{"azp": clientDeService, "scope": porteeReserver}, b.intrus),
	}
	for nom, brut := range cas {
		t.Run(nom, func(t *testing.T) {
			if _, err := service.Verify(t.Context(), brut); !errors.Is(err, token.ErrInvalidToken) {
				t.Fatalf("ErrInvalidToken attendu, obtenu %v", err)
			}
		})
	}
}

func TestVerificateurDOrigineInchangeParForService(t *testing.T) {
	b := nouveauBanc(t)
	b.verificateurDeService(t)
	if _, err := b.verificateur.Verify(t.Context(), b.jeton(sansModification)); err != nil {
		t.Fatalf("jeton d'utilisateur refusé par le vérificateur d'origine : %v", err)
	}
	if _, err := b.verificateur.Verify(t.Context(), b.jetonDeService(sansModification)); err == nil {
		t.Fatalf("jeton de service accepté par le vérificateur des utilisateurs")
	}
}

type appelDelegue struct {
	autorisation []string
	acheteur     []string
}

func (b *banc) delegation(t *testing.T) http.Handler {
	t.Helper()
	delegation := token.Delegation{Service: b.verificateurDeService(t), User: b.verificateur, UserRole: roleAcheteur}
	atteint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		revendications, ok := token.ClaimsFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.Header().Set("X-Sujet", revendications.Subject)
		w.Header().Set("X-Roles", strings.Join(revendications.Roles, ","))
		w.WriteHeader(http.StatusNoContent)
	})
	return delegation.Authenticate(slog.New(slog.NewTextHandler(io.Discard, nil)), atteint)
}

func appelerDelegue(t *testing.T, gestionnaire http.Handler, appel appelDelegue) *httptest.ResponseRecorder {
	t.Helper()
	requete := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/interne/v1/articles/x/reservations", nil)
	for _, valeur := range appel.autorisation {
		requete.Header.Add("Authorization", valeur)
	}
	for _, valeur := range appel.acheteur {
		requete.Header.Add(token.UserTokenHeader, valeur)
	}
	enregistreur := httptest.NewRecorder()
	gestionnaire.ServeHTTP(enregistreur, requete)
	return enregistreur
}

func TestDelegationAcceptee(t *testing.T) {
	b := nouveauBanc(t)
	acheteur := b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, charge(roleAcheteur), b.cle)
	r := appelerDelegue(t, b.delegation(t), appelDelegue{
		autorisation: []string{"Bearer " + b.jetonDeService(sansModification)},
		acheteur:     []string{acheteur},
	})
	if r.Code != http.StatusNoContent {
		t.Fatalf("statut %d, 204 attendu : %s", r.Code, r.Body)
	}
	if r.Header().Get("X-Sujet") != sujet || r.Header().Get("X-Roles") != roleAcheteur {
		t.Fatalf("revendications du contexte : sujet %q, rôles %q ; celles de l'acheteur attendues", r.Header().Get("X-Sujet"), r.Header().Get("X-Roles"))
	}
}

func TestDelegationRefusee(t *testing.T) {
	b := nouveauBanc(t)
	gestionnaire := b.delegation(t)
	service := "Bearer " + b.jetonDeService(sansModification)
	acheteur := b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, charge(roleAcheteur), b.cle)
	autreAcheteur := b.jeton(remplacer("sub", "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"))
	cas := map[string]struct {
		appel   appelDelegue
		attendu int
	}{
		"jeton de service absent":                     {appelDelegue{acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton de service en Basic":                   {appelDelegue{autorisation: []string{"Basic YTpi"}, acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton de service illisible":                  {appelDelegue{autorisation: []string{"Bearer pas-un-jeton"}, acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton de service d'un autre client":          {appelDelegue{autorisation: []string{"Bearer " + b.jetonDeService(remplacer("azp", "intrus"))}, acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton de service sans la portée":             {appelDelegue{autorisation: []string{"Bearer " + b.jetonDeService(retirer("scope"))}, acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton de service expiré":                     {appelDelegue{autorisation: []string{"Bearer " + b.jetonDeService(remplacer("exp", int64(1)))}, acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton d'utilisateur à la place du service":   {appelDelegue{autorisation: []string{"Bearer " + acheteur}, acheteur: []string{acheteur}}, http.StatusUnauthorized},
		"jeton de l'acheteur absent":                  {appelDelegue{autorisation: []string{service}}, http.StatusForbidden},
		"jeton de l'acheteur vide":                    {appelDelegue{autorisation: []string{service}, acheteur: []string{""}}, http.StatusForbidden},
		"jeton de l'acheteur répété":                  {appelDelegue{autorisation: []string{service}, acheteur: []string{acheteur, autreAcheteur}}, http.StatusForbidden},
		"jeton de l'acheteur répété à l'identique":    {appelDelegue{autorisation: []string{service}, acheteur: []string{acheteur, acheteur}}, http.StatusForbidden},
		"jeton de l'acheteur préfixé Bearer":          {appelDelegue{autorisation: []string{service}, acheteur: []string{"Bearer " + acheteur}}, http.StatusForbidden},
		"jeton de l'acheteur illisible":               {appelDelegue{autorisation: []string{service}, acheteur: []string{"pas-un-jeton"}}, http.StatusForbidden},
		"jeton de l'acheteur signé par une autre clé": {appelDelegue{autorisation: []string{service}, acheteur: []string{b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, charge(roleAcheteur), b.intrus)}}, http.StatusForbidden},
		"jeton de l'acheteur d'un autre client":       {appelDelegue{autorisation: []string{service}, acheteur: []string{b.jeton(remplacer("azp", "intrus"))}}, http.StatusForbidden},
		"jeton de service à la place de l'acheteur":   {appelDelegue{autorisation: []string{service}, acheteur: []string{b.jetonDeService(sansModification)}}, http.StatusForbidden},
		"jeton de l'acheteur sans le rôle acheteur":   {appelDelegue{autorisation: []string{service}, acheteur: []string{b.signer(map[string]any{"alg": "RS256", "kid": "k1"}, charge("vendeur"), b.cle)}}, http.StatusForbidden},
		"jeton de l'acheteur expiré":                  {appelDelegue{autorisation: []string{service}, acheteur: []string{b.jeton(remplacer("exp", int64(1)))}}, http.StatusForbidden},
		"jeton de l'acheteur d'une autre audience":    {appelDelegue{autorisation: []string{service}, acheteur: []string{b.jeton(remplacer("aud", "transactions"))}}, http.StatusForbidden},
		"jeton de l'acheteur algorithme none":         {appelDelegue{autorisation: []string{service}, acheteur: []string{b64([]byte(`{"alg":"none"}`)) + "." + b64([]byte(`{"sub":"x"}`)) + "."}}, http.StatusForbidden},
		"jeton de service absent, acheteur absent":    {appelDelegue{}, http.StatusUnauthorized},
		"jeton de service refusé, acheteur sans rôle": {appelDelegue{autorisation: []string{"Bearer " + acheteur}, acheteur: []string{"pas-un-jeton"}}, http.StatusUnauthorized},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			r := appelerDelegue(t, gestionnaire, c.appel)
			if r.Code != c.attendu {
				t.Fatalf("statut %d, %d attendu : %s", r.Code, c.attendu, r.Body)
			}
			if r.Header().Get("X-Sujet") != "" {
				t.Fatalf("gestionnaire atteint malgré le refus")
			}
			if c.attendu == http.StatusUnauthorized && r.Header().Get("WWW-Authenticate") == "" {
				t.Fatalf("401 sans WWW-Authenticate")
			}
		})
	}
}

func TestCredentialMasqueEnAffichageEtJournal(t *testing.T) {
	brut := strings.Join([]string{"eyJ", "charge-confidentielle", "signature"}, ".")
	credential := token.Credential(brut)
	for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
		if sortie := fmt.Sprintf(format, credential); strings.Contains(sortie, brut) {
			t.Errorf("%s révèle le jeton : %s", format, sortie)
		}
	}
	if sortie := fmt.Sprintf("%v %+v %#v", struct{ C token.Credential }{credential}, struct{ C token.Credential }{credential}, struct{ C token.Credential }{credential}); strings.Contains(sortie, brut) {
		t.Errorf("structure englobante révèle le jeton : %s", sortie)
	}
	var journal bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&journal, nil))
	logger.Info("appel", slog.Any("jeton", credential), slog.Group("requete", slog.Any("jeton", credential)))
	logger.Info("appel", "jeton", credential)
	slog.New(slog.NewTextHandler(&journal, nil)).Info("appel", "jeton", credential)
	if strings.Contains(journal.String(), brut) {
		t.Errorf("journal révèle le jeton : %s", journal.String())
	}
	var ligne map[string]any
	if err := json.Unmarshal(bytes.SplitN(journal.Bytes(), []byte("\n"), 2)[0], &ligne); err != nil || ligne["jeton"] == "" || ligne["jeton"] == nil {
		t.Errorf("valeur masquée absente du journal : %v %v", ligne, err)
	}
	if credential.Reveal() != brut {
		t.Errorf("Reveal rend %q, %q attendu", credential.Reveal(), brut)
	}
}

func TestCredentialDuContexteApresAuthentification(t *testing.T) {
	b := nouveauBanc(t)
	brut := b.jeton(sansModification)
	var recu token.Credential
	var trouve bool
	protege := b.verificateur.Authenticate(slog.New(slog.NewTextHandler(io.Discard, nil)), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recu, trouve = token.CredentialFrom(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	if code := appeler(t, protege, "Bearer "+brut); code != http.StatusNoContent || !trouve || recu.Reveal() != brut {
		t.Fatalf("jeton du contexte : statut %d, trouvé %t, identique %t", code, trouve, recu.Reveal() == brut)
	}
	if _, present := token.CredentialFrom(t.Context()); present {
		t.Fatalf("jeton trouvé dans un contexte vide")
	}
}
