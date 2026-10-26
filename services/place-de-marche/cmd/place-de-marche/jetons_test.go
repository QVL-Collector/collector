package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	emetteurAttendu  = "https://auth.collector.test/realms/collector"
	audienceAttendue = "place-de-marche"
	clientAutorise   = "collector-web"
	roleVendeur      = "vendeur"
	roleAcheteur     = "acheteur"
)

type cleDeSignature struct {
	privee *rsa.PrivateKey
	kid    string
}

type revendications map[string]any

type modification func(entete, charge map[string]any)

type fabriqueDeJetons struct {
	t      testing.TB
	cle    cleDeSignature
	intrus cleDeSignature
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

func servirJWKS(t testing.TB, cles ...cleDeSignature) *httptest.Server {
	t.Helper()
	jeu := make([]map[string]string, 0, len(cles))
	for _, c := range cles {
		jeu = append(jeu, c.jwk())
	}
	contenu, err := json.Marshal(map[string]any{"keys": jeu})
	if err != nil {
		t.Fatalf("encodage du JWKS : %v", err)
	}
	serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(contenu)
	}))
	t.Cleanup(serveur.Close)
	return serveur
}

func (f *fabriqueDeJetons) encoderJSON(valeur any) string {
	f.t.Helper()
	octets, err := json.Marshal(valeur)
	if err != nil {
		f.t.Fatalf("encodage JSON : %v", err)
	}
	return encoder(octets)
}

func (f *fabriqueDeJetons) signerRS256(entete, charge map[string]any, cle cleDeSignature) string {
	f.t.Helper()
	signe := f.encoderJSON(entete) + "." + f.encoderJSON(charge)
	empreinte := sha256.Sum256([]byte(signe))
	signature, err := rsa.SignPKCS1v15(rand.Reader, cle.privee, crypto.SHA256, empreinte[:])
	if err != nil {
		f.t.Fatalf("signature : %v", err)
	}
	return signe + "." + encoder(signature)
}

func (f *fabriqueDeJetons) signerHS256(entete, charge map[string]any) string {
	f.t.Helper()
	signe := f.encoderJSON(entete) + "." + f.encoderJSON(charge)
	mac := hmac.New(sha256.New, f.cle.privee.N.Bytes())
	_, _ = mac.Write([]byte(signe))
	return signe + "." + encoder(mac.Sum(nil))
}

func (f *fabriqueDeJetons) nonSigne(entete, charge map[string]any) string {
	f.t.Helper()
	return f.encoderJSON(entete) + "." + f.encoderJSON(charge) + "."
}

func revendicationsValides(sujet string, roles ...string) revendications {
	maintenant := time.Now()
	return revendications{
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

func (f *fabriqueDeJetons) entete() map[string]any {
	return map[string]any{"alg": "RS256", "typ": "JWT", "kid": f.cle.kid}
}

func (f *fabriqueDeJetons) emettre(sujet string, roles []string, modifications ...modification) string {
	f.t.Helper()
	entete := f.entete()
	charge := map[string]any(revendicationsValides(sujet, roles...))
	for _, m := range modifications {
		m(entete, charge)
	}
	return f.signerRS256(entete, charge, f.cle)
}

func (f *fabriqueDeJetons) vendeur(sujet string) string {
	return f.emettre(sujet, []string{roleVendeur})
}

func (f *fabriqueDeJetons) acheteur(sujet string) string {
	return f.emettre(sujet, []string{roleAcheteur})
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
