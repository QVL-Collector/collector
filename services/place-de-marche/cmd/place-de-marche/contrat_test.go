package main

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
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
	contrat, err := chargeur.LoadFromFile(filepath.Join("..", "..", "..", "..", "contracts", "api", "place-de-marche", "v1.openapi.yaml"))
	if err != nil {
		t.Fatalf("chargement du contrat : %v", err)
	}
	if err := contrat.Validate(t.Context()); err != nil {
		t.Fatalf("contrat invalide : %v", err)
	}
	contrat.Servers = openapi3.Servers{{URL: base + "/marche/v1"}}
	routeur, err := gorillamux.NewRouter(contrat)
	if err != nil {
		t.Fatalf("routeur du contrat : %v", err)
	}
	return routeur
}

func jouerTousLesStatuts(t *testing.T, e *environnement) {
	t.Helper()
	jeton := e.vendeur()
	e.categories(t)
	article := e.creer(t, jeton, corpsComplet("Contrat", e.categories(t)[0].ID, 1200, 300))
	e.creer(t, jeton, `{}`)
	e.mesArticles(t, jeton, "")
	e.mesArticles(t, jeton, "?page=1&taille=1")
	e.relire(t, jeton, article.ID)
	e.modifier(t, jeton, article.ID, `{"revision":1,"titre":"Contrat modifié"}`).exiger(t, http.StatusOK)
	e.modifier(t, jeton, article.ID, `{"revision":1,"titre":"Périmé"}`).exiger(t, http.StatusConflict)
	e.lireArticle(t, e.vendeur(), article.ID).exiger(t, http.StatusNotFound)
	e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(`{"prix":{"montantCentimes":-1,"devise":"EUR"}}`)).exiger(t, http.StatusBadRequest)
	e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(`{"titre":"a","titre":"b"}`)).exiger(t, http.StatusBadRequest)
	e.envoyer(t, http.MethodGet, cheminArticles+"?taille=51", jeton).exiger(t, http.StatusBadRequest)
	e.envoyer(t, http.MethodGet, cheminArticles, "").exiger(t, http.StatusUnauthorized)
	e.envoyer(t, http.MethodGet, cheminArticles, bearer(e.jetons.acheteur(uuid.NewString()))).exiger(t, http.StatusForbidden)
	e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsBrut(corpsDeTaille(limiteCorps+1))).exiger(t, http.StatusRequestEntityTooLarge)
	e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(`{}`), typeDeContenu("text/plain")).exiger(t, http.StatusUnsupportedMediaType)
}

func verifierEchange(t *testing.T, routeur routers.Router, ech echange) {
	t.Helper()
	route, parametres, err := routeur.FindRoute(ech.requete)
	if err != nil {
		t.Fatalf("%s %s absent du contrat : %v", ech.requete.Method, ech.requete.URL, err)
	}
	options := &openapi3filter.Options{IncludeResponseStatus: true, AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}
	entree := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    ech.requete,
			PathParams: parametres,
			Route:      route,
			Options:    options,
		},
		Status:  ech.reponse.statut,
		Header:  ech.reponse.entetes,
		Body:    io.NopCloser(bytes.NewReader(ech.reponse.corps)),
		Options: options,
	}
	if err := openapi3filter.ValidateResponse(t.Context(), entree); err != nil {
		t.Errorf("%s %s → %d non conforme au contrat : %v\n%s", ech.requete.Method, ech.requete.URL.Path, ech.reponse.statut, err, ech.reponse.corps)
	}
}

func TestUS009ReponsesConformesAuContratOpenAPI(t *testing.T) {
	e := demarrerEnMemoire(t)
	routeur := chargerContrat(t, e.adresse)
	jouerTousLesStatuts(t, e)
	for _, ech := range e.historique() {
		verifierEchange(t, routeur, ech)
	}
}
