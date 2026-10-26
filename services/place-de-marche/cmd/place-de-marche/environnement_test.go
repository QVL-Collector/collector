package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/api"
)

const (
	cheminCategories = "/marche/v1/categories"
	cheminArticles   = "/marche/v1/vendeur/articles"
	typeJSON         = "application/json"
	typeProbleme     = "application/problem+json"
)

type echange struct {
	requete *http.Request
	reponse reponse
}

type environnement struct {
	serveur  *httptest.Server
	jetons   *fabriqueDeJetons
	mu       sync.Mutex
	echanges []echange
}

type reponse struct {
	statut  int
	entetes http.Header
	corps   []byte
}

type envoi struct {
	corps       []byte
	typeContenu string
}

type option func(*envoi)

type montant struct {
	MontantCentimes int64  `json:"montantCentimes"`
	Devise          string `json:"devise"`
}

type categorieRecue struct {
	ID      string `json:"id"`
	Libelle string `json:"libelle"`
}

type articleRecu struct {
	ID          string          `json:"id"`
	Etat        string          `json:"etat"`
	Revision    int             `json:"revision"`
	Titre       string          `json:"titre"`
	Description string          `json:"description"`
	Categorie   *categorieRecue `json:"categorie"`
	Prix        *montant        `json:"prix"`
	FraisDePort *montant        `json:"fraisDePort"`
	CreeLe      string          `json:"creeLe"`
	ModifieLe   string          `json:"modifieLe"`
}

type pageArticles struct {
	Elements []articleRecu `json:"elements"`
	Page     int           `json:"page"`
	Taille   int           `json:"taille"`
	Total    int           `json:"total"`
}

type listeCategories struct {
	Elements []categorieRecue `json:"elements"`
}

type erreurDeChamp struct {
	Champ string `json:"champ"`
	Code  string `json:"code"`
}

type probleme struct {
	Type    string          `json:"type"`
	Title   string          `json:"title"`
	Status  int             `json:"status"`
	Detail  string          `json:"detail"`
	Erreurs []erreurDeChamp `json:"erreurs"`
}

func journalMuet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func demarrer(t *testing.T, service api.ArticleService) *environnement {
	t.Helper()
	cle := nouvelleCle(t, "cle-collector")
	jwks := servirJWKS(t, cle)
	verificateur, err := token.NewVerifier(t.Context(), token.Config{
		Issuer:          emetteurAttendu,
		Audience:        audienceAttendue,
		AuthorizedParty: clientAutorise,
		KeySetURL:       jwks.URL,
	})
	if err != nil {
		t.Fatalf("vérificateur de jetons : %v", err)
	}
	serveur := httptest.NewServer(publicHandler(journalMuet(), service, verificateur))
	t.Cleanup(serveur.Close)
	return &environnement{
		serveur: serveur,
		jetons:  &fabriqueDeJetons{t: t, cle: cle, intrus: nouvelleCle(t, cle.kid)},
	}
}

func demarrerEnMemoire(t *testing.T) *environnement {
	t.Helper()
	stockage := nouveauStockageEnMemoire(jeuDeCategoriesFictif)
	return demarrer(t, annonces.NewService(stockage, stockage))
}

func corpsJSON(corps string) option {
	return func(e *envoi) {
		e.corps = []byte(corps)
		e.typeContenu = typeJSON
	}
}

func corpsBrut(corps []byte) option {
	return func(e *envoi) {
		e.corps = corps
		e.typeContenu = typeJSON
	}
}

func typeDeContenu(valeur string) option {
	return func(e *envoi) {
		e.typeContenu = valeur
	}
}

func (e *environnement) envoyer(t *testing.T, methode, chemin, jeton string, options ...option) reponse {
	t.Helper()
	parametres := envoi{}
	for _, o := range options {
		o(&parametres)
	}
	requete, err := http.NewRequestWithContext(t.Context(), methode, e.serveur.URL+chemin, bytes.NewReader(parametres.corps))
	if err != nil {
		t.Fatalf("requête %s %s : %v", methode, chemin, err)
	}
	if parametres.typeContenu != "" {
		requete.Header.Set("Content-Type", parametres.typeContenu)
	}
	if jeton != "" {
		requete.Header.Set("Authorization", jeton)
	}
	resultat := e.executer(t, requete)
	requete.Body = io.NopCloser(bytes.NewReader(parametres.corps))
	e.mu.Lock()
	e.echanges = append(e.echanges, echange{requete: requete, reponse: resultat})
	e.mu.Unlock()
	return resultat
}

func (e *environnement) executer(t *testing.T, requete *http.Request) reponse {
	t.Helper()
	brute, err := e.serveur.Client().Do(requete)
	if err != nil {
		t.Fatalf("appel %s %s : %v", requete.Method, requete.URL.Path, err)
	}
	defer func() { _ = brute.Body.Close() }()
	corps, err := io.ReadAll(brute.Body)
	if err != nil {
		t.Fatalf("lecture de la réponse : %v", err)
	}
	return reponse{statut: brute.StatusCode, entetes: brute.Header, corps: corps}
}

func bearer(jeton string) string {
	return "Bearer " + jeton
}

func (e *environnement) vendeur() string {
	return bearer(e.jetons.vendeur(uuid.NewString()))
}

func (r reponse) exiger(t *testing.T, statut int) reponse {
	t.Helper()
	if r.statut != statut {
		t.Fatalf("statut %d attendu, %d reçu : %s", statut, r.statut, r.corps)
	}
	return r
}

func (r reponse) lire(t *testing.T, cible any) {
	t.Helper()
	if err := json.Unmarshal(r.corps, cible); err != nil {
		t.Fatalf("réponse illisible : %v : %s", err, r.corps)
	}
}

func (r reponse) probleme(t *testing.T, statut int) probleme {
	t.Helper()
	r.exiger(t, statut)
	if !strings.HasPrefix(r.entetes.Get("Content-Type"), typeProbleme) {
		t.Fatalf("Content-Type %q reçu, %q attendu", r.entetes.Get("Content-Type"), typeProbleme)
	}
	var p probleme
	r.lire(t, &p)
	if p.Status != statut || p.Title == "" || p.Type == "" {
		t.Fatalf("Problem Details incomplet : %s", r.corps)
	}
	return p
}

func (e *environnement) creer(t *testing.T, jeton, corps string) articleRecu {
	t.Helper()
	var article articleRecu
	e.envoyer(t, http.MethodPost, cheminArticles, jeton, corpsJSON(corps)).exiger(t, http.StatusCreated).lire(t, &article)
	return article
}

func (e *environnement) lireArticle(t *testing.T, jeton, id string) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodGet, cheminArticles+"/"+id, jeton)
}

func (e *environnement) relire(t *testing.T, jeton, id string) articleRecu {
	t.Helper()
	var article articleRecu
	e.lireArticle(t, jeton, id).exiger(t, http.StatusOK).lire(t, &article)
	return article
}

func (e *environnement) modifier(t *testing.T, jeton, id, corps string) reponse {
	t.Helper()
	return e.envoyer(t, http.MethodPut, cheminArticles+"/"+id, jeton, corpsJSON(corps))
}

func (e *environnement) mesArticles(t *testing.T, jeton, requete string) pageArticles {
	t.Helper()
	var page pageArticles
	e.envoyer(t, http.MethodGet, cheminArticles+requete, jeton).exiger(t, http.StatusOK).lire(t, &page)
	return page
}

func (e *environnement) categories(t *testing.T) []categorieRecue {
	t.Helper()
	var liste listeCategories
	e.envoyer(t, http.MethodGet, cheminCategories, "").exiger(t, http.StatusOK).lire(t, &liste)
	if len(liste.Elements) == 0 {
		t.Fatal("aucune catégorie proposée")
	}
	return liste.Elements
}

func (e *environnement) historique() []echange {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]echange(nil), e.echanges...)
}

func contexteBorne(t *testing.T, duree time.Duration) context.Context {
	t.Helper()
	ctx, annuler := context.WithTimeout(t.Context(), duree)
	t.Cleanup(annuler)
	return ctx
}
