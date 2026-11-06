package main

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/google/uuid"
)

func chargerContrat(t *testing.T, base string) routers.Router {
	t.Helper()
	chargeur := openapi3.NewLoader()
	contrat, err := chargeur.LoadFromFile(filepath.Join("..", "..", "..", "..", "contracts", "api", "interactions", "v1.openapi.yaml"))
	if err != nil {
		t.Fatalf("chargement du contrat : %v", err)
	}
	if err := contrat.Validate(t.Context()); err != nil {
		t.Fatalf("contrat invalide : %v", err)
	}
	contrat.Servers = openapi3.Servers{{URL: base + "/interactions/v1"}}
	routeur, err := gorillamux.NewRouter(contrat)
	if err != nil {
		t.Fatalf("routeur du contrat : %v", err)
	}
	return routeur
}

func verifierEchange(t *testing.T, routeur routers.Router, r reponseHTTP) {
	t.Helper()
	route, parametres, err := routeur.FindRoute(r.requete)
	if err != nil {
		t.Fatalf("%s %s absent du contrat : %v", r.requete.Method, r.requete.URL, err)
	}
	options := &openapi3filter.Options{IncludeResponseStatus: true, AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}
	entree := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: r.requete, PathParams: parametres, Route: route, Options: options},
		Status:                 r.statut,
		Header:                 r.entetes,
		Body:                   io.NopCloser(bytes.NewReader(r.corps)),
		Options:                options,
	}
	if err := openapi3filter.ValidateResponse(t.Context(), entree); err != nil {
		t.Errorf("%s %s → %d non conforme au contrat : %v\n%s", r.requete.Method, r.requete.URL, r.statut, err, r.corps)
	}
}

func (b *bancInteractions) reponsesConformesAuContrat(t *testing.T) {
	vendeur := uuid.NewString()
	jeton := b.jetons.vendeur(vendeur)
	b.lister(t, jeton, "")
	b.lister(t, jeton, "?page=2&taille=1")
	b.lister(t, jeton, "?taille=51").exiger(t, http.StatusBadRequest)
	b.lister(t, jeton, "?page=0").exiger(t, http.StatusBadRequest)
	b.lister(t, "", "").exiger(t, http.StatusUnauthorized)
	b.lister(t, b.jetons.emettre(vendeur, []string{roleAcheteur}), "").exiger(t, http.StatusForbidden)
	routeur := chargerContrat(t, b.adresse)
	for _, r := range b.historique {
		if strings.HasPrefix(r.requete.URL.Path, cheminNotifications) {
			verifierEchange(t, routeur, r)
		}
	}
}

func (r reponseHTTP) exiger(t *testing.T, statut int) reponseHTTP {
	t.Helper()
	if r.statut != statut {
		t.Fatalf("statut %d attendu, %d reçu : %s", statut, r.statut, r.corps)
	}
	return r
}
