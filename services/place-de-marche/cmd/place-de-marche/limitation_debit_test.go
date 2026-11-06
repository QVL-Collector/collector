package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/ratelimit"
	"collector/services/place-de-marche/internal/annonces"
)

const (
	rafaleDeTest     = 3
	rafaleDeTirTest  = 20
	adresseDuClient  = "203.0.113.7"
	adresseDeTirTest = "198.51.100.42"
)

type routeLimitee struct {
	methode string
	chemin  func(id string) string
	corps   string
}

func routesLimitees() map[string]routeLimitee {
	return map[string]routeLimitee{
		"soumission": {http.MethodPost, cheminSoumission, `{"revision":1}`},
		"modification": {http.MethodPut, func(id string) string { return cheminArticles + "/" + id },
			corpsRevision(1, corpsComplet("Figurine modifiée", categorieSure, 100, 0))},
	}
}

func politiqueDeTest() ratelimit.Policy {
	return ratelimit.Policy{
		Nominal:         ratelimit.PerMinute(1, rafaleDeTest),
		LoadTest:        ratelimit.PerMinute(1, rafaleDeTirTest),
		LoadTestAddress: netip.MustParseAddr(adresseDeTirTest),
	}
}

func demarrerLimite(t *testing.T) *environnement {
	t.Helper()
	stockage := nouveauStockageEnMemoire(jeuDeCategoriesFictif)
	limiteur := ratelimit.NewGuard(politiqueDeTest(), journalMuet(), time.Now)
	return demarrerAvecLimiteur(t, annonces.NewService(stockage, stockage), journalMuet(), limiteur)
}

func (e *environnement) depuis(t *testing.T, forwardedFor, methode, chemin, jeton, corps string) reponse {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), methode, e.adresse+chemin, bytes.NewReader([]byte(corps)))
	if err != nil {
		t.Fatalf("requête %s %s : %v", methode, chemin, err)
	}
	if corps != "" {
		requete.Header.Set("Content-Type", typeJSON)
	}
	if jeton != "" {
		requete.Header.Set("Authorization", jeton)
	}
	if forwardedFor != "" {
		requete.Header.Set("X-Forwarded-For", forwardedFor)
	}
	resultat := e.executer(t, requete)
	requete.Body = io.NopCloser(bytes.NewReader([]byte(corps)))
	e.mu.Lock()
	e.echanges = append(e.echanges, echange{requete: requete, reponse: resultat})
	e.mu.Unlock()
	return resultat
}

func (e *environnement) appeler(t *testing.T, route routeLimitee, forwardedFor, jeton, id string) reponse {
	t.Helper()
	return e.depuis(t, forwardedFor, route.methode, route.chemin(id), jeton, route.corps)
}

func (r reponse) exigerServie(t *testing.T) {
	t.Helper()
	if r.statut == http.StatusTooManyRequests || r.statut >= http.StatusInternalServerError {
		t.Fatalf("requête servie attendue, %d reçu : %s", r.statut, r.corps)
	}
}

func (r reponse) exigerTropDeRequetes(t *testing.T) {
	t.Helper()
	r.probleme(t, http.StatusTooManyRequests)
	attente, err := strconv.Atoi(r.entetes.Get("Retry-After"))
	if err != nil || attente < 1 {
		t.Fatalf("Retry-After entier ≥ 1 attendu, %q reçu", r.entetes.Get("Retry-After"))
	}
}

func (e *environnement) brouillonDe(t *testing.T, jeton string) string {
	t.Helper()
	return e.creer(t, jeton, corpsComplet("Figurine", categorieSure, 100, 0)).ID
}

func adresseNumero(i int) string {
	return fmt.Sprintf("192.0.2.%d", i+1)
}

func TestUS027AC1MemeAdresseEtMemeCompteObtiennent429AvecRetryAfter(t *testing.T) {
	for nom, route := range routesLimitees() {
		t.Run(nom, func(t *testing.T) {
			e := demarrerLimite(t)
			vendeur := e.vendeur()
			id := e.brouillonDe(t, vendeur)
			for range rafaleDeTest {
				e.appeler(t, route, adresseDuClient, vendeur, id).exigerServie(t)
			}

			e.appeler(t, route, adresseDuClient, vendeur, id).exigerTropDeRequetes(t)
		})
	}
}

func TestUS027AC1MemeCompteDepuisDesAdressesDifferentesObtient429(t *testing.T) {
	for nom, route := range routesLimitees() {
		t.Run(nom, func(t *testing.T) {
			e := demarrerLimite(t)
			vendeur := e.vendeur()
			id := e.brouillonDe(t, vendeur)
			for i := range rafaleDeTest {
				e.appeler(t, route, adresseNumero(i), vendeur, id).exigerServie(t)
			}

			e.appeler(t, route, adresseNumero(rafaleDeTest), vendeur, id).exigerTropDeRequetes(t)
		})
	}
}

func TestUS027AC1MemeAdressePourDesComptesDifferentsObtient429(t *testing.T) {
	for nom, route := range routesLimitees() {
		t.Run(nom, func(t *testing.T) {
			e := demarrerLimite(t)
			for range rafaleDeTest {
				vendeur := e.vendeur()
				e.appeler(t, route, adresseDuClient, vendeur, e.brouillonDe(t, vendeur)).exigerServie(t)
			}

			vendeur := e.vendeur()
			e.appeler(t, route, adresseDuClient, vendeur, e.brouillonDe(t, vendeur)).exigerTropDeRequetes(t)
		})
	}
}

func TestUS027AC1AutreAdresseEtAutreCompteNonPenalises(t *testing.T) {
	e := demarrerLimite(t)
	route := routesLimitees()["soumission"]
	abuseur := e.vendeur()
	id := e.brouillonDe(t, abuseur)
	for range rafaleDeTest {
		e.appeler(t, route, adresseDuClient, abuseur, id).exigerServie(t)
	}
	e.appeler(t, route, adresseDuClient, abuseur, id).exigerTropDeRequetes(t)

	honnete := e.vendeur()
	e.appeler(t, route, "203.0.113.99", honnete, e.brouillonDe(t, honnete)).exiger(t, http.StatusAccepted)
}

func TestUS027AC1RequetesNonAuthentifieesOuSansRoleNeConsommentPasLaLimite(t *testing.T) {
	e := demarrerLimite(t)
	route := routesLimitees()["soumission"]
	id := uuid.NewString()
	acheteur := bearer(e.jetons.acheteur(uuid.NewString()))
	for range 3 * rafaleDeTest {
		e.appeler(t, route, adresseDuClient, "", id).probleme(t, http.StatusUnauthorized)
		e.appeler(t, route, adresseDuClient, acheteur, id).probleme(t, http.StatusForbidden)
	}

	vendeur := e.vendeur()
	brouillon := e.brouillonDe(t, vendeur)
	for range rafaleDeTest {
		e.appeler(t, route, adresseDuClient, vendeur, brouillon).exigerServie(t)
	}
	e.appeler(t, route, adresseDuClient, vendeur, brouillon).exigerTropDeRequetes(t)
}

func TestUS027AC4ForwardedForForgeResteA429SurLaPlaceDeMarche(t *testing.T) {
	forges := map[string]func(i int) string{
		"valeur unique": func(i int) string { return adresseNumero(i) + ", " + adresseDuClient },
		"liste":         func(i int) string { return adresseNumero(i) + ", 10.0.0.1, 127.0.0.1, " + adresseDuClient },
		"adresse de tir forgée": func(int) string {
			return adresseDeTirTest + ", " + adresseDuClient
		},
	}
	for nom, forge := range forges {
		t.Run(nom, func(t *testing.T) {
			e := demarrerLimite(t)
			route := routesLimitees()["soumission"]
			for i := range rafaleDeTest {
				vendeur := e.vendeur()
				e.appeler(t, route, forge(i), vendeur, e.brouillonDe(t, vendeur)).exigerServie(t)
			}

			vendeur := e.vendeur()
			e.appeler(t, route, forge(rafaleDeTest), vendeur, e.brouillonDe(t, vendeur)).exigerTropDeRequetes(t)
		})
	}
}

func TestUS027AC3AdresseDeTirRelevePuisLimiteSurLaPlaceDeMarche(t *testing.T) {
	e := demarrerLimite(t)
	route := routesLimitees()["soumission"]
	vendeur := e.vendeur()
	id := e.brouillonDe(t, vendeur)
	for range rafaleDeTirTest {
		e.appeler(t, route, adresseDeTirTest, vendeur, id).exigerServie(t)
	}

	e.appeler(t, route, adresseDeTirTest, vendeur, id).exigerTropDeRequetes(t)
}

func TestUS027CatalogueEtAutresRoutesNonLimites(t *testing.T) {
	e := demarrerLimite(t)
	vendeur := e.vendeur()
	id := e.brouillonDe(t, vendeur)
	appels := map[string]func() reponse{
		"catalogue": func() reponse {
			return e.depuis(t, adresseDuClient, http.MethodGet, cheminCatalogue, "", "")
		},
		"catégories": func() reponse {
			return e.depuis(t, adresseDuClient, http.MethodGet, cheminCategories, "", "")
		},
		"mes articles": func() reponse {
			return e.depuis(t, adresseDuClient, http.MethodGet, cheminArticles, vendeur, "")
		},
		"lecture d'un article": func() reponse {
			return e.depuis(t, adresseDuClient, http.MethodGet, cheminArticles+"/"+id, vendeur, "")
		},
		"création de brouillon": func() reponse {
			return e.depuis(t, adresseDuClient, http.MethodPost, cheminArticles, vendeur, corpsComplet("Figurine", categorieSure, 100, 0))
		},
	}
	for nom, appel := range appels {
		t.Run(nom, func(t *testing.T) {
			for i := range 10 * rafaleDeTest {
				if r := appel(); r.statut == http.StatusTooManyRequests || r.statut >= http.StatusBadRequest {
					t.Fatalf("appel %d : %d reçu : %s", i+1, r.statut, r.corps)
				}
			}
		})
	}
}

func TestUS027Reponses429ConformesAuContratOpenAPI(t *testing.T) {
	e := demarrerLimite(t)
	routeur := chargerContrat(t, e.adresse)
	vendeur := e.vendeur()
	id := e.brouillonDe(t, vendeur)
	for _, route := range routesLimitees() {
		for range rafaleDeTest + 1 {
			e.appeler(t, route, adresseNumero(0), vendeur, id)
		}
	}
	trouves := 0
	for _, ech := range e.historique() {
		if ech.reponse.statut == http.StatusTooManyRequests {
			trouves++
		}
		verifierEchange(t, routeur, ech)
	}
	if trouves < 2 {
		t.Fatalf("%d réponses 429 vérifiées, au moins 2 attendues", trouves)
	}
}
