package marche_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/logging"
	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/commandes/marche"
	"collector/services/transactions/internal/montants"
	"collector/services/transactions/internal/problemes"
)

const (
	jetonAcheteur = "eyJhbGciOiJSUzI1NiJ9.acheteur.signature"
	typeProbleme  = "application/problem+json"
)

type requeteRecue struct {
	methode string
	chemin  string
	entetes http.Header
	corps   []byte
}

type placeDeMarcheSimulee struct {
	mu      sync.Mutex
	recues  []requeteRecue
	statut  int
	type_   string
	corps   string
	serveur *httptest.Server
}

func simuler(t *testing.T, statut int, typeDeContenu, corps string) *placeDeMarcheSimulee {
	t.Helper()
	p := &placeDeMarcheSimulee{statut: statut, type_: typeDeContenu, corps: corps}
	p.serveur = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		octets, _ := io.ReadAll(r.Body)
		p.mu.Lock()
		p.recues = append(p.recues, requeteRecue{methode: r.Method, chemin: r.URL.Path, entetes: r.Header.Clone(), corps: octets})
		p.mu.Unlock()
		if p.type_ != "" {
			w.Header().Set("Content-Type", p.type_)
		}
		w.WriteHeader(p.statut)
		_, _ = io.WriteString(w, p.corps)
	}))
	t.Cleanup(p.serveur.Close)
	return p
}

const titreDeReference = "Montre de gousset 1920"

func reservationJSON(article, devise string) string {
	return reservationTitreeJSON(article, devise, titreDeReference)
}

func reservationTitreeJSON(article, devise, titre string) string {
	return fmt.Sprintf(`{"reservationId":"6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b","articleId":%q,"titre":%q,"echeance":"2026-11-16T10:25:00.123456Z","prix":{"montantCentimes":12000,"devise":%q},"fraisDePort":{"montantCentimes":590,"devise":"EUR"}}`, article, titre, devise)
}

func probleme(urn string, statut int) string {
	return fmt.Sprintf(`{"type":%q,"title":"refus","status":%d}`, urn, statut)
}

func nouvelArticle(t *testing.T) commandes.ArticleID {
	t.Helper()
	article, err := commandes.ParseArticleID(uuid.NewString())
	if err != nil {
		t.Fatalf("article : %v", err)
	}
	return article
}

func reserver(t *testing.T, p *placeDeMarcheSimulee, article commandes.ArticleID) (commandes.Reservation, error) {
	t.Helper()
	client := marche.NewClient(&http.Client{Timeout: 2 * time.Second}, p.serveur.URL)
	ctx := logging.WithCorrelationID(t.Context(), "correlation-us030")
	return client.Reserve(ctx, article, token.Credential(jetonAcheteur))
}

func TestReservationCreeeOuReprise(t *testing.T) {
	for _, statut := range []int{http.StatusCreated, http.StatusOK} {
		t.Run(http.StatusText(statut), func(t *testing.T) {
			article := nouvelArticle(t)
			p := simuler(t, statut, "application/json", reservationJSON(article.String(), "EUR"))
			reservation, err := reserver(t, p, article)
			if err != nil {
				t.Fatalf("réservation refusée : %v", err)
			}
			echeance := time.Date(2026, 11, 16, 10, 25, 0, 123456000, time.UTC)
			if reservation.ID.String() != "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b" || reservation.Article != article || !reservation.ExpiresAt.Equal(echeance) {
				t.Fatalf("réservation inattendue : %+v", reservation)
			}
			if reservation.Price != (montants.Money{Cents: 12000, Currency: montants.Euro}) || reservation.ShippingFee != (montants.Money{Cents: 590, Currency: montants.Euro}) {
				t.Fatalf("montants inattendus : %+v %+v", reservation.Price, reservation.ShippingFee)
			}
		})
	}
}

func TestRequeteEnvoyeeALaPlaceDeMarche(t *testing.T) {
	article := nouvelArticle(t)
	p := simuler(t, http.StatusCreated, "application/json", reservationJSON(article.String(), "EUR"))
	if _, err := reserver(t, p, article); err != nil {
		t.Fatalf("réservation : %v", err)
	}
	if len(p.recues) != 1 {
		t.Fatalf("%d requêtes, 1 attendue", len(p.recues))
	}
	r := p.recues[0]
	if r.methode != http.MethodPost || r.chemin != "/interne/v1/articles/"+article.String()+"/reservations" {
		t.Fatalf("%s %s, POST /interne/v1/articles/%s/reservations attendu", r.methode, r.chemin, article)
	}
	if valeurs := r.entetes.Values(token.UserTokenHeader); len(valeurs) != 1 || valeurs[0] != jetonAcheteur {
		t.Fatalf("%s %v, jeton brut de l'acheteur attendu une fois", token.UserTokenHeader, valeurs)
	}
	if r.entetes.Get(logging.CorrelationHeader) != "correlation-us030" {
		t.Fatalf("corrélation %q non reprise", r.entetes.Get(logging.CorrelationHeader))
	}
	if !strings.Contains(r.entetes.Get("Accept"), "application/json") {
		t.Fatalf("Accept %q", r.entetes.Get("Accept"))
	}
	if len(r.corps) != 0 {
		t.Fatalf("corps envoyé : %s", r.corps)
	}
	if r.entetes.Get("Authorization") != "" {
		t.Fatalf("le client ajoute lui-même un Authorization : %q", r.entetes.Get("Authorization"))
	}
}

func TestRefusMetierTraduits(t *testing.T) {
	cas := map[string]struct {
		statut  int
		corps   string
		attendu error
	}{
		"422 non réservable":       {http.StatusUnprocessableEntity, probleme(problemes.ArticleNotReservable, 422), commandes.ErrArticleNotReservable},
		"409 déjà réservé":         {http.StatusConflict, probleme(problemes.ArticleAlreadyReserved, 409), commandes.ErrArticleAlreadyReserved},
		"409 réservation active":   {http.StatusConflict, probleme(problemes.ReservationActive, 409), commandes.ErrBuyerAlreadyHoldsAnother},
		"409 de type inconnu":      {http.StatusConflict, probleme("urn:collector:probleme:autre", 409), commandes.ErrMarketplaceUnavailable},
		"409 sans corps":           {http.StatusConflict, "", commandes.ErrMarketplaceUnavailable},
		"401":                      {http.StatusUnauthorized, probleme("about:blank", 401), commandes.ErrMarketplaceUnavailable},
		"403":                      {http.StatusForbidden, probleme("about:blank", 403), commandes.ErrMarketplaceUnavailable},
		"404":                      {http.StatusNotFound, probleme("about:blank", 404), commandes.ErrMarketplaceUnavailable},
		"500":                      {http.StatusInternalServerError, probleme("about:blank", 500), commandes.ErrMarketplaceUnavailable},
		"502":                      {http.StatusBadGateway, "", commandes.ErrMarketplaceUnavailable},
		"503":                      {http.StatusServiceUnavailable, "", commandes.ErrMarketplaceUnavailable},
		"302":                      {http.StatusFound, "", commandes.ErrMarketplaceUnavailable},
		"201 illisible":            {http.StatusCreated, `{"reservationId":`, commandes.ErrMarketplaceUnavailable},
		"201 vide":                 {http.StatusCreated, ``, commandes.ErrMarketplaceUnavailable},
		"201 sans échéance":        {http.StatusCreated, `{"reservationId":"6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b","articleId":"ART","titre":"Montre","prix":{"montantCentimes":1,"devise":"EUR"},"fraisDePort":{"montantCentimes":0,"devise":"EUR"}}`, commandes.ErrMarketplaceUnavailable},
		"201 sans prix":            {http.StatusCreated, `{"reservationId":"6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b","articleId":"ART","titre":"Montre","echeance":"2026-11-16T10:25:00Z","fraisDePort":{"montantCentimes":0,"devise":"EUR"}}`, commandes.ErrMarketplaceUnavailable},
		"201 réservation non UUID": {http.StatusCreated, `{"reservationId":"x","articleId":"ART","titre":"Montre","echeance":"2026-11-16T10:25:00Z","prix":{"montantCentimes":1,"devise":"EUR"},"fraisDePort":{"montantCentimes":0,"devise":"EUR"}}`, commandes.ErrMarketplaceUnavailable},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			article := nouvelArticle(t)
			typeDeContenu := typeProbleme
			if c.statut < 300 {
				typeDeContenu = "application/json"
			}
			p := simuler(t, c.statut, typeDeContenu, strings.ReplaceAll(c.corps, "ART", article.String()))
			if _, err := reserver(t, p, article); !errors.Is(err, c.attendu) {
				t.Fatalf("erreur %v, %v attendue", err, c.attendu)
			}
		})
	}
	t.Run("201 en devise autre qu'EUR", func(t *testing.T) {
		article := nouvelArticle(t)
		p := simuler(t, http.StatusCreated, "application/json", reservationJSON(article.String(), "USD"))
		if _, err := reserver(t, p, article); !errors.Is(err, commandes.ErrMarketplaceUnavailable) {
			t.Fatalf("erreur %v, indisponibilité attendue", err)
		}
	})
}

type transportEnPanne struct{}

func (transportEnPanne) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("jeton de service non obtenu")
}

func TestPlaceDeMarcheInjoignableOuJetonNonObtenu(t *testing.T) {
	article := nouvelArticle(t)
	cas := map[string]*marche.Client{
		"échec du jeton de service": marche.NewClient(&http.Client{Transport: transportEnPanne{}}, "http://place-de-marche-interne:8081"),
		"adresse injoignable":       marche.NewClient(&http.Client{Timeout: time.Second}, "http://127.0.0.1:1"),
	}
	for nom, client := range cas {
		t.Run(nom, func(t *testing.T) {
			if _, err := client.Reserve(t.Context(), article, token.Credential(jetonAcheteur)); !errors.Is(err, commandes.ErrMarketplaceUnavailable) {
				t.Fatalf("erreur %v, indisponibilité attendue", err)
			}
		})
	}
	t.Run("délai dépassé", func(t *testing.T) {
		lent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(3 * time.Second):
			}
			w.WriteHeader(http.StatusCreated)
		}))
		t.Cleanup(lent.Close)
		client := marche.NewClient(&http.Client{Timeout: 200 * time.Millisecond}, lent.URL)
		if _, err := client.Reserve(t.Context(), article, token.Credential(jetonAcheteur)); !errors.Is(err, commandes.ErrMarketplaceUnavailable) {
			t.Fatalf("erreur %v, indisponibilité attendue", err)
		}
	})
}
