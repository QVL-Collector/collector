package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/services/interactions/internal/notifications"
)

const cheminNotifications = "/interactions/v1/notifications"

var champsDUneNotification = []string{"id", "articleId", "revision", "issue", "motif", "decideLe"}

type listeEnMemoire struct {
	mu         sync.Mutex
	parVendeur map[notifications.SellerID][]notifications.Notification
	demandes   []notifications.SellerID
}

func (l *listeEnMemoire) ListForSeller(_ context.Context, vendeur notifications.SellerID, page notifications.PageRequest) (notifications.Page, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.demandes = append(l.demandes, vendeur)
	siennes := l.parVendeur[vendeur]
	return notifications.Page{Notifications: siennes, Total: len(siennes)}, nil
}

func (l *listeEnMemoire) vendeursDemandes() []notifications.SellerID {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.demandes)
}

type apiEnMemoire struct {
	adresse string
	jetons  *fabriqueDeJetons
	liste   *listeEnMemoire
}

func journalMuet() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func demarrerAPIEnMemoire(t *testing.T) *apiEnMemoire {
	t.Helper()
	jetons := nouvelleFabriqueDeJetons(t)
	liste := &listeEnMemoire{parVendeur: map[notifications.SellerID][]notifications.Notification{}}
	serveur := httptest.NewServer(publicHandler(journalMuet(), liste, jetons.verificateur()))
	t.Cleanup(serveur.Close)
	return &apiEnMemoire{adresse: serveur.URL, jetons: jetons, liste: liste}
}

func (a *apiEnMemoire) notifier(t *testing.T, sujet string, issue notifications.Issue, motif *notifications.Reason) notifications.Notification {
	t.Helper()
	vendeur, err := notifications.ParseSellerID(sujet)
	if err != nil {
		t.Fatalf("sujet %s : %v", sujet, err)
	}
	article, err := notifications.ParseArticleID(uuid.NewString())
	if err != nil {
		t.Fatalf("article : %v", err)
	}
	n := notifications.Notification{
		ID:        notifications.NotificationID(uuid.New()),
		Article:   article,
		Revision:  2,
		Issue:     issue,
		Reason:    motif,
		DecidedAt: time.Date(2026, 11, 6, 9, 15, 2, 0, time.UTC),
	}
	a.liste.parVendeur[vendeur] = append(a.liste.parVendeur[vendeur], n)
	return n
}

type reponseHTTP struct {
	requete *http.Request
	statut  int
	entetes http.Header
	corps   []byte
}

func appeler(t *testing.T, adresse, autorisation, parametres string) reponseHTTP {
	t.Helper()
	requete, err := http.NewRequestWithContext(t.Context(), http.MethodGet, adresse+cheminNotifications+parametres, nil)
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	if autorisation != "" {
		requete.Header.Set("Authorization", autorisation)
	}
	reponse, err := http.DefaultClient.Do(requete)
	if err != nil {
		t.Fatalf("appel de la liste : %v", err)
	}
	defer func() { _ = reponse.Body.Close() }()
	corps, err := io.ReadAll(reponse.Body)
	if err != nil {
		t.Fatalf("lecture de la réponse : %v", err)
	}
	return reponseHTTP{requete: requete, statut: reponse.StatusCode, entetes: reponse.Header, corps: corps}
}

type notificationRecue struct {
	ID        string  `json:"id"`
	ArticleID string  `json:"articleId"`
	Revision  int     `json:"revision"`
	Issue     string  `json:"issue"`
	Motif     *string `json:"motif"`
	DecideLe  string  `json:"decideLe"`
}

type pageRecue struct {
	Elements []notificationRecue `json:"elements"`
	Page     int                 `json:"page"`
	Taille   int                 `json:"taille"`
	Total    int                 `json:"total"`
}

func (r reponseHTTP) page(t *testing.T) pageRecue {
	t.Helper()
	if r.statut != http.StatusOK {
		t.Fatalf("statut %d attendu 200 : %s", r.statut, r.corps)
	}
	var p pageRecue
	if err := json.Unmarshal(r.corps, &p); err != nil {
		t.Fatalf("réponse illisible : %v : %s", err, r.corps)
	}
	return p
}

func identifiantsDe(p pageRecue) []string {
	ids := make([]string, 0, len(p.Elements))
	for _, n := range p.Elements {
		ids = append(ids, n.ID)
	}
	slices.Sort(ids)
	return ids
}

func identifiantsAttendus(ns ...notifications.Notification) []string {
	ids := make([]string, 0, len(ns))
	for _, n := range ns {
		ids = append(ids, n.ID.String())
	}
	slices.Sort(ids)
	return ids
}

func motifCode(code string) *notifications.Reason {
	motif := notifications.Reason(code)
	return &motif
}

func TestUS028AC6_ChaqueVendeurNeLitQueSesNotifications(t *testing.T) {
	a := demarrerAPIEnMemoire(t)
	vendeurA, vendeurB := uuid.NewString(), uuid.NewString()
	deA := []notifications.Notification{
		a.notifier(t, vendeurA, notifications.IssuePublished, nil),
		a.notifier(t, vendeurA, notifications.IssueRefused, motifCode("article_interdit")),
	}
	deB := a.notifier(t, vendeurB, notifications.IssueToVerify, motifCode("prix_a_confirmer"))

	pageA := appeler(t, a.adresse, a.jetons.vendeur(vendeurA), "").page(t)
	pageB := appeler(t, a.adresse, a.jetons.vendeur(vendeurB), "").page(t)

	if got, want := identifiantsDe(pageA), identifiantsAttendus(deA...); !slices.Equal(got, want) {
		t.Fatalf("vendeur A lit %v, attendu %v", got, want)
	}
	if got, want := identifiantsDe(pageB), identifiantsAttendus(deB); !slices.Equal(got, want) {
		t.Fatalf("vendeur B lit %v, attendu %v", got, want)
	}
	demandes := a.liste.vendeursDemandes()
	if len(demandes) != 2 || demandes[0].String() != vendeurA || demandes[1].String() != vendeurB {
		t.Fatalf("liste lue pour %v, attendu le sujet de chaque jeton (%s puis %s)", demandes, vendeurA, vendeurB)
	}
}

func TestUS028AC6_VendeurSansNotificationLitUneListeVide(t *testing.T) {
	a := demarrerAPIEnMemoire(t)
	a.notifier(t, uuid.NewString(), notifications.IssuePublished, nil)

	page := appeler(t, a.adresse, a.jetons.vendeur(uuid.NewString()), "").page(t)

	if len(page.Elements) != 0 || page.Total != 0 {
		t.Fatalf("vendeur sans notification lit %+v", page)
	}
}

func TestUS028AC6_SansJetonOuJetonInvalideRepond401SansLireLaListe(t *testing.T) {
	a := demarrerAPIEnMemoire(t)
	victime := uuid.NewString()
	a.notifier(t, victime, notifications.IssueRefused, motifCode("prix_abusif"))
	for nom, autorisation := range a.jetons.jetonsInvalides(uuid.NewString(), victime) {
		t.Run(nom, func(t *testing.T) {
			r := appeler(t, a.adresse, autorisation, "")
			if r.statut != http.StatusUnauthorized {
				t.Fatalf("statut %d, 401 attendu : %s", r.statut, r.corps)
			}
		})
	}
	if demandes := a.liste.vendeursDemandes(); len(demandes) != 0 {
		t.Fatalf("liste lue %d fois pour des appels non authentifiés", len(demandes))
	}
}

func TestUS028AC6_JetonSansLeRoleVendeurNeLitAucuneNotification(t *testing.T) {
	a := demarrerAPIEnMemoire(t)
	sujet := uuid.NewString()
	a.notifier(t, sujet, notifications.IssuePublished, nil)
	for nom, roles := range map[string][]string{"acheteur": {roleAcheteur}, "admin": {roleAdmin}, "aucun rôle": {}} {
		t.Run(nom, func(t *testing.T) {
			r := appeler(t, a.adresse, a.jetons.emettre(sujet, roles), "")
			if r.statut != http.StatusForbidden {
				t.Fatalf("statut %d, 403 attendu : %s", r.statut, r.corps)
			}
		})
	}
	if demandes := a.liste.vendeursDemandes(); len(demandes) != 0 {
		t.Fatalf("liste lue %d fois sans le rôle vendeur", len(demandes))
	}
}

func TestUS028AC4_NotificationNExposeQueDesCodesEtDesIdentifiants(t *testing.T) {
	a := demarrerAPIEnMemoire(t)
	vendeur := uuid.NewString()
	publiee := a.notifier(t, vendeur, notifications.IssuePublished, nil)
	refusee := a.notifier(t, vendeur, notifications.IssueRefused, motifCode("coordonnees_detectees"))

	r := appeler(t, a.adresse, a.jetons.vendeur(vendeur), "")

	exigerSeulementLesChampsDuContrat(t, r.corps, 2)
	for _, n := range r.page(t).Elements {
		switch n.ID {
		case publiee.ID.String():
			if n.Issue != "publie" || n.Motif != nil || n.ArticleID != publiee.Article.String() || n.Revision != 2 {
				t.Fatalf("notification publiée %+v", n)
			}
		case refusee.ID.String():
			if n.Issue != "refuse" || n.Motif == nil || *n.Motif != "coordonnees_detectees" {
				t.Fatalf("notification refusée %+v", n)
			}
		default:
			t.Fatalf("notification inattendue %+v", n)
		}
		if _, err := time.Parse(time.RFC3339, n.DecideLe); err != nil {
			t.Fatalf("decideLe %q hors RFC 3339", n.DecideLe)
		}
	}
}

func exigerSeulementLesChampsDuContrat(t *testing.T, corps []byte, nombre int) []map[string]any {
	t.Helper()
	var brute struct {
		Elements []map[string]any `json:"elements"`
	}
	if err := json.Unmarshal(corps, &brute); err != nil || len(brute.Elements) != nombre {
		t.Fatalf("réponse inattendue : %v : %s", err, corps)
	}
	attendus := slices.Sorted(slices.Values(champsDUneNotification))
	for _, element := range brute.Elements {
		if champs := slices.Sorted(maps.Keys(element)); !slices.Equal(champs, attendus) {
			t.Fatalf("champs %v, attendus %v", champs, attendus)
		}
	}
	return brute.Elements
}
