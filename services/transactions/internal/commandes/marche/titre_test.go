package marche_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"collector/services/transactions/internal/commandes"
)

func TestUS032TitreDeLaReservationLu(t *testing.T) {
	cas := map[string]string{
		"titre courant":  titreDeReference,
		"un caractère":   "A",
		"200 caractères": strings.Repeat("a", 200),
		"200 caractères accentués (400 octets)": strings.Repeat("é", 200),
		"guillemets et symboles":                `Carte « rare » n°7 — "édition" 1999 & co`,
	}
	for nom, titre := range cas {
		t.Run(nom, func(t *testing.T) {
			article := nouvelArticle(t)
			p := simuler(t, http.StatusCreated, "application/json", reservationTitreeJSON(article.String(), "EUR", titre))
			reservation, err := reserver(t, p, article)
			if err != nil {
				t.Fatalf("réservation refusée : %v", err)
			}
			if reservation.Title != titre {
				t.Fatalf("titre %q, %q attendu", reservation.Title, titre)
			}
		})
	}
}

func TestUS032TitreHorsContratRefuseCommePlaceDeMarcheIndisponible(t *testing.T) {
	sansTitre := `{"reservationId":"6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b","articleId":"ART","echeance":"2026-11-16T10:25:00Z","prix":{"montantCentimes":1,"devise":"EUR"},"fraisDePort":{"montantCentimes":0,"devise":"EUR"}}`
	cas := map[string]func(article string) string{
		"titre absent":            func(article string) string { return strings.ReplaceAll(sansTitre, "ART", article) },
		"titre vide":              func(article string) string { return reservationTitreeJSON(article, "EUR", "") },
		"titre de 201 caractères": func(article string) string { return reservationTitreeJSON(article, "EUR", strings.Repeat("a", 201)) },
		"titre de 201 caractères accentués": func(article string) string {
			return reservationTitreeJSON(article, "EUR", strings.Repeat("é", 201))
		},
		"titre nul": func(article string) string {
			return strings.Replace(reservationTitreeJSON(article, "EUR", "x"), `"titre":"x"`, `"titre":null`, 1)
		},
		"titre d'un autre type": func(article string) string {
			return strings.Replace(reservationTitreeJSON(article, "EUR", "x"), `"titre":"x"`, `"titre":42`, 1)
		},
	}
	for nom, corps := range cas {
		t.Run(nom, func(t *testing.T) {
			article := nouvelArticle(t)
			p := simuler(t, http.StatusCreated, "application/json", corps(article.String()))
			if _, err := reserver(t, p, article); !errors.Is(err, commandes.ErrMarketplaceUnavailable) {
				t.Fatalf("erreur %v, indisponibilité attendue", err)
			}
		})
	}
}
