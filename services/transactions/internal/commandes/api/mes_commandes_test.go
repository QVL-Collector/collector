package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/services/transactions/internal/commandes"
)

const typeSaisieInvalide = "urn:collector:probleme:saisie-invalide"

type pageRecue struct {
	Elements []commandeRecue `json:"elements"`
	Page     int             `json:"page"`
	Taille   int             `json:"taille"`
	Total    int             `json:"total"`
}

type erreurDeChampRecue struct {
	Champ string `json:"champ"`
	Code  string `json:"code"`
}

type problemeDePagination struct {
	Type    string               `json:"type"`
	Status  int                  `json:"status"`
	Erreurs []erreurDeChampRecue `json:"erreurs"`
}

func (b *banc) lister(t *testing.T, sujet, requete string) reponse {
	t.Helper()
	return b.appeler(t, http.MethodGet, cheminCommand+requete, b.emetteur.acheteur(sujet), "")
}

func (b *banc) derniereListe(t *testing.T) appelListe {
	t.Helper()
	if len(b.commandes.listes) == 0 {
		t.Fatalf("service de liste non appelé")
	}
	return b.commandes.listes[len(b.commandes.listes)-1]
}

func TestUS032AC1ListeAvecArticleMontantEcheanceEtEtat(t *testing.T) {
	b := nouveauBanc(t)
	sujet, acheteur, article := identifiants(t)
	enAttente := commandeDe(t, acheteur, article)
	expiree := commandeDe(t, acheteur, article)
	expiree.Title = "Montre de gousset 1920"
	expiree.Status = commandes.StatusExpired
	b.commandes.page = commandes.OrderPage{Orders: []commandes.Order{enAttente, expiree}, Total: 2}
	r := b.lister(t, sujet, "")
	if r.statut != http.StatusOK || !strings.HasPrefix(r.entetes.Get("Content-Type"), "application/json") {
		t.Fatalf("statut %d, type %q : %s", r.statut, r.entetes.Get("Content-Type"), r.corps)
	}
	var page pageRecue
	lire(t, r, &page)
	if page.Page != 1 || page.Taille != 20 || page.Total != 2 || len(page.Elements) != 2 {
		t.Fatalf("page %+v ; page 1, taille 20, total 2, 2 éléments attendus", page)
	}
	premier, second := page.Elements[0], page.Elements[1]
	if premier.ID != enAttente.ID.String() || second.ID != expiree.ID.String() {
		t.Fatalf("ordre %s, %s ; ordre du service attendu", premier.ID, second.ID)
	}
	if premier.Titre != "Figurine de collection" || second.Titre != "Montre de gousset 1920" {
		t.Errorf("titres %q et %q", premier.Titre, second.Titre)
	}
	if premier.Statut != "en_attente_paiement" || second.Statut != "expiree" {
		t.Errorf("états %q et %q", premier.Statut, second.Statut)
	}
	if premier.TotalPaye != (montantRecu{12590, "EUR"}) || premier.Echeance != "2026-11-16T10:25:00.123456Z" || premier.ArticleID != article.String() {
		t.Errorf("élément %+v", premier)
	}
}

func TestUS032AC1ListeVideNonNulle(t *testing.T) {
	b := nouveauBanc(t)
	sujet, _, _ := identifiants(t)
	b.commandes.page = commandes.OrderPage{Total: 0}
	r := b.lister(t, sujet, "")
	var brut map[string]json.RawMessage
	lire(t, r, &brut)
	if r.statut != http.StatusOK || string(brut["elements"]) != "[]" || string(brut["total"]) != "0" {
		t.Fatalf("statut %d : %s ; elements [] et total 0 attendus", r.statut, r.corps)
	}
}

func TestUS032AC3ListeDemandeePourLeSujetDuJeton(t *testing.T) {
	b := nouveauBanc(t)
	sujetB, acheteurB, _ := identifiants(t)
	r := b.appeler(t, http.MethodGet, cheminCommand+"?acheteurId="+uuid.NewString(), b.emetteur.acheteur(sujetB), "")
	if r.statut != http.StatusOK {
		t.Fatalf("statut %d : %s", r.statut, r.corps)
	}
	if appel := b.derniereListe(t); appel.acheteur != acheteurB {
		t.Fatalf("liste demandée pour %v, acheteur du jeton %v attendu", appel.acheteur, acheteurB)
	}
}

func TestUS032PaginationAcceptee(t *testing.T) {
	cas := map[string]commandes.PageRequest{
		"":                         {Number: 1, Size: 20},
		"?page=1":                  {Number: 1, Size: 20},
		"?taille=1":                {Number: 1, Size: 1},
		"?taille=50":               {Number: 1, Size: 50},
		"?page=10000":              {Number: 10000, Size: 20},
		"?page=2&taille=5":         {Number: 2, Size: 5},
		"?page=10000&taille=50":    {Number: 10000, Size: 50},
		"?taille=7&page=3":         {Number: 3, Size: 7},
		"?page=2&taille=5&autre=x": {Number: 2, Size: 5},
	}
	for requete, attendue := range cas {
		t.Run("requête "+requete, func(t *testing.T) {
			b := nouveauBanc(t)
			sujet, _, _ := identifiants(t)
			r := b.lister(t, sujet, requete)
			var page pageRecue
			lire(t, r, &page)
			if r.statut != http.StatusOK || page.Page != attendue.Number || page.Taille != attendue.Size {
				t.Fatalf("statut %d, page %d, taille %d ; 200, %d, %d attendus", r.statut, page.Page, page.Taille, attendue.Number, attendue.Size)
			}
			if appel := b.derniereListe(t); appel.page != attendue {
				t.Fatalf("page demandée au service %+v, %+v attendue", appel.page, attendue)
			}
		})
	}
}

func (b *banc) refusDePagination(t *testing.T, requete string) problemeDePagination {
	t.Helper()
	sujet, _, _ := identifiants(t)
	r := b.lister(t, sujet, requete)
	var p problemeDePagination
	lire(t, r, &p)
	if r.statut != http.StatusBadRequest || p.Type != typeSaisieInvalide || r.entetes.Get("Content-Type") != typeProbleme {
		t.Fatalf("statut %d type %q ; 400 %s attendu : %s", r.statut, p.Type, typeSaisieInvalide, r.corps)
	}
	return p
}

func TestUS032PaginationRefuseeEn400(t *testing.T) {
	cas := map[string]erreurDeChampRecue{
		"?page=0":            {"page", "hors_limites"},
		"?page=-1":           {"page", "hors_limites"},
		"?page=10001":        {"page", "hors_limites"},
		"?taille=0":          {"taille", "hors_limites"},
		"?taille=51":         {"taille", "hors_limites"},
		"?taille=-3":         {"taille", "hors_limites"},
		"?page=":             {"page", "invalide"},
		"?taille=":           {"taille", "invalide"},
		"?page=abc":          {"page", "invalide"},
		"?taille=1.5":        {"taille", "invalide"},
		"?page=1&page=2":     {"page", "invalide"},
		"?taille=5&taille=5": {"taille", "invalide"},
	}
	for requete, attendue := range cas {
		t.Run("requête "+requete, func(t *testing.T) {
			b := nouveauBanc(t)
			p := b.refusDePagination(t, requete)
			if !slices.Contains(p.Erreurs, attendue) {
				t.Errorf("erreurs %+v sans %+v", p.Erreurs, attendue)
			}
			if len(b.commandes.listes) != 0 {
				t.Fatalf("service appelé malgré une pagination invalide")
			}
		})
	}
	t.Run("page au-delà des entiers 64 bits", func(t *testing.T) {
		p := nouveauBanc(t).refusDePagination(t, "?page=99999999999999999999")
		if len(p.Erreurs) != 1 || p.Erreurs[0].Champ != "page" {
			t.Fatalf("erreurs %+v ; une erreur sur page attendue", p.Erreurs)
		}
		t.Logf("page au-delà des entiers 64 bits : code %q", p.Erreurs[0].Code)
	})
	t.Run("page et taille invalides ensemble", func(t *testing.T) {
		p := nouveauBanc(t).refusDePagination(t, "?page=0&taille=abc")
		if len(p.Erreurs) != 2 {
			t.Fatalf("erreurs %+v ; une par champ attendue", p.Erreurs)
		}
		t.Logf("deux champs invalides : %+v", p.Erreurs)
	})
}

func TestUS032ListeSansAcheteurAuthentifie(t *testing.T) {
	b := nouveauBanc(t)
	sujet, _, _ := identifiants(t)
	cas := map[string]struct {
		jeton  string
		statut int
	}{
		"sans jeton":            {"", http.StatusUnauthorized},
		"jeton illisible":       {"pas-un-jeton", http.StatusUnauthorized},
		"expiré":                {b.emetteur.jeton(sujet, func(r map[string]any) { r["exp"] = int64(1) }, "acheteur"), http.StatusUnauthorized},
		"autre audience":        {b.emetteur.jeton(sujet, func(r map[string]any) { r["aud"] = "place-de-marche" }, "acheteur"), http.StatusUnauthorized},
		"sans le rôle acheteur": {b.emetteur.jeton(sujet, func(map[string]any) {}, "vendeur"), http.StatusForbidden},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			if r := b.appeler(t, http.MethodGet, cheminCommand, c.jeton, ""); r.statut != c.statut {
				t.Fatalf("statut %d, %d attendu", r.statut, c.statut)
			}
		})
	}
	if len(b.commandes.listes) != 0 {
		t.Fatalf("service de liste appelé sans acheteur authentifié")
	}
}

func TestUS032ErreurDuServiceDeListeSansDetailTechnique(t *testing.T) {
	b := nouveauBanc(t)
	sujet, _, _ := identifiants(t)
	b.commandes.erreurListe = errors.New("base perdue")
	r := b.lister(t, sujet, "")
	if r.statut != http.StatusInternalServerError || strings.Contains(string(r.corps), "base perdue") {
		t.Fatalf("statut %d : %s ; 500 sans détail attendu", r.statut, r.corps)
	}
}

func TestUS032TitreExposeALaCreationEtAuDetail(t *testing.T) {
	b := nouveauBanc(t)
	sujet, acheteur, article := identifiants(t)
	commande := commandeDe(t, acheteur, article)
	commande.Title = "Carte « rare » n°7 — édition 1999"
	b.commandes.resultat = commandes.Recorded{Order: commande, Created: true}
	b.commandes.commandes[commande.ID] = commande
	for nom, r := range map[string]reponse{
		"création": b.appeler(t, http.MethodPost, cheminCommand, b.emetteur.acheteur(sujet), `{"articleId":"`+article.String()+`"}`),
		"détail":   b.appeler(t, http.MethodGet, cheminCommand+"/"+commande.ID.String(), b.emetteur.acheteur(sujet), ""),
	} {
		var recue commandeRecue
		lire(t, r, &recue)
		if recue.Titre != commande.Title {
			t.Errorf("%s : titre %q, %q attendu", nom, recue.Titre, commande.Title)
		}
	}
}

func TestUS032AC3DetailDUneCommandeDAutruiEn404CommeInexistante(t *testing.T) {
	b := nouveauBanc(t)
	_, acheteurA, article := identifiants(t)
	sujetB, _, _ := identifiants(t)
	commande := commandeDe(t, acheteurA, article)
	commande.ExpiresAt = time.Now().Add(time.Hour)
	b.commandes.commandes[commande.ID] = commande
	deA := b.appeler(t, http.MethodGet, cheminCommand+"/"+commande.ID.String(), b.emetteur.acheteur(sujetB), "")
	inexistante := b.appeler(t, http.MethodGet, cheminCommand+"/"+uuid.NewString(), b.emetteur.acheteur(sujetB), "")
	if deA.statut != http.StatusNotFound || string(deA.corps) != string(inexistante.corps) {
		t.Fatalf("détail de A lu par B : %d %s ; 404 identique à une commande inexistante attendu (%s)", deA.statut, deA.corps, inexistante.corps)
	}
	if strings.Contains(string(deA.corps), commande.Title) || strings.Contains(string(deA.corps), article.String()) {
		t.Fatalf("404 qui divulgue la commande de A : %s", deA.corps)
	}
}
