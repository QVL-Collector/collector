package commandes_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/commandes/marche"
)

const titreDeLaMontre = "Montre de gousset 1920"

func (b *bancDuService) reserverTitreJusquA(t *testing.T, article commandes.ArticleID, titre string, echeance time.Time) {
	t.Helper()
	_, _, reservation := nouveauxIdentifiants(t)
	b.marche.reservations[article] = commandes.Reservation{ID: reservation, Article: article, Title: titre, ExpiresAt: echeance, Price: euros(8000), ShippingFee: euros(500)}
}

func (b *bancDuService) commandesDe(t *testing.T, acheteur commandes.BuyerID, page commandes.PageRequest) commandes.OrderPage {
	t.Helper()
	lues, err := b.service.OwnOrders(t.Context(), acheteur, page)
	if err != nil {
		t.Fatalf("liste des commandes : %v", err)
	}
	return lues
}

func premierePage() commandes.PageRequest {
	return commandes.PageRequest{Number: 1, Size: commandes.DefaultPageSize}
}

func TestUS032AC1TitreDeLaReservationFigeDansLaCommande(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	b.reserverTitreJusquA(t, article, titreDeLaMontre, echeanceDeReference)
	creee := b.premiereCommande(t, acheteur, article)
	if creee.Order.Title != titreDeLaMontre {
		t.Fatalf("titre de la commande créée %q, %q attendu", creee.Order.Title, titreDeLaMontre)
	}
	if len(b.registre.enregistrements) != 1 || b.registre.enregistrements[0].Reservation.Title != titreDeLaMontre {
		t.Fatalf("titre transmis à l'enregistrement : %+v", b.registre.enregistrements)
	}
	lue, err := b.service.OwnOrder(t.Context(), acheteur, creee.Order.ID)
	if err != nil || lue.Title != titreDeLaMontre {
		t.Fatalf("détail : %+v, %v ; titre %q attendu", lue, err, titreDeLaMontre)
	}
	page := b.commandesDe(t, acheteur, premierePage())
	if len(page.Orders) != 1 || page.Orders[0].Title != titreDeLaMontre {
		t.Fatalf("liste : %+v ; titre %q attendu", page.Orders, titreDeLaMontre)
	}
}

func TestUS032AC1ListeAvecMontantEcheanceEtEtat(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	b.reserverTitreJusquA(t, article, titreDeLaMontre, echeanceDeReference)
	creee := b.premiereCommande(t, acheteur, article)
	page := b.commandesDe(t, acheteur, premierePage())
	if page.Total != 1 || len(page.Orders) != 1 {
		t.Fatalf("page %+v, une commande attendue", page)
	}
	lue := page.Orders[0]
	if lue.ID != creee.Order.ID || lue.Amounts.TotalPaid != euros(8500) || !lue.ExpiresAt.Equal(echeanceDeReference) || lue.Status != commandes.StatusAwaitingPayment {
		t.Fatalf("commande listée %+v ; total 85 €, échéance %s, en attente attendus", lue, echeanceDeReference)
	}
}

func TestUS032AC2ListeRelueApresLEcheanceCommandeExpiree(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	b.reserverTitreJusquA(t, article, titreDeLaMontre, echeanceDeReference)
	b.premiereCommande(t, acheteur, article)
	etats := []struct {
		instant time.Time
		attendu commandes.Status
	}{
		{echeanceDeReference.Add(-time.Nanosecond), commandes.StatusAwaitingPayment},
		{echeanceDeReference, commandes.StatusExpired},
		{echeanceDeReference.Add(time.Hour), commandes.StatusExpired},
	}
	for _, etat := range etats {
		horloge.regler(etat.instant)
		page := b.commandesDe(t, acheteur, premierePage())
		if len(page.Orders) != 1 || page.Orders[0].Status != etat.attendu {
			t.Fatalf("liste relue à %s : %+v ; statut %q attendu", etat.instant, page.Orders, etat.attendu)
		}
		if !page.Orders[0].ExpiresAt.Equal(echeanceDeReference) || page.Orders[0].Amounts.TotalPaid != euros(8500) {
			t.Fatalf("échéance ou montant changés par la relecture : %+v", page.Orders[0])
		}
	}
}

func TestUS032AC2SeuleLaCommandeEchueBasculeEnExpiree(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	_, autreArticle, _ := nouveauxIdentifiants(t)
	b.reserverTitreJusquA(t, article, "Première", echeanceDeReference)
	premiere := b.premiereCommande(t, acheteur, article)
	b.reserverTitreJusquA(t, autreArticle, "Seconde", echeanceDeReference.Add(10*time.Minute))
	seconde := b.premiereCommande(t, acheteur, autreArticle)
	horloge.regler(echeanceDeReference.Add(time.Second))
	statuts := map[commandes.ID]commandes.Status{}
	for _, c := range b.commandesDe(t, acheteur, premierePage()).Orders {
		statuts[c.ID] = c.Status
	}
	if statuts[premiere.Order.ID] != commandes.StatusExpired || statuts[seconde.Order.ID] != commandes.StatusAwaitingPayment {
		t.Fatalf("statuts %v ; première expiree, seconde en_attente_paiement attendues", statuts)
	}
}

func TestUS032AC3ListeDemandeePourLAcheteurAuthentifieSeulement(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteurA, article := nouveauBancHorloge(t, horloge)
	acheteurB, _, _ := nouveauxIdentifiants(t)
	b.reserverTitreJusquA(t, article, titreDeLaMontre, echeanceDeReference)
	b.premiereCommande(t, acheteurA, article)
	page := b.commandesDe(t, acheteurB, premierePage())
	if page.Total != 0 || len(page.Orders) != 0 {
		t.Fatalf("B voit %+v, aucune commande attendue", page)
	}
	derniere := b.registre.listes[len(b.registre.listes)-1]
	if derniere.acheteur != acheteurB {
		t.Fatalf("liste demandée pour %v, %v attendu", derniere.acheteur, acheteurB)
	}
}

func TestUS032PageDemandeeTransmiseTelleQuelle(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, _, _ := nouveauxIdentifiants(t)
	for _, page := range []commandes.PageRequest{{Number: 1, Size: 1}, {Number: 3, Size: 7}, {Number: commandes.MaxPageNumber, Size: commandes.MaxPageSize}} {
		b.commandesDe(t, acheteur, page)
		if derniere := b.registre.listes[len(b.registre.listes)-1]; derniere.page != page {
			t.Fatalf("page transmise %+v, %+v attendue", derniere.page, page)
		}
	}
}

func TestUS032BornesDePaginationAuContrat(t *testing.T) {
	if commandes.DefaultPageSize != 20 || commandes.MaxPageSize != 50 || commandes.MaxPageNumber != 10_000 {
		t.Fatalf("bornes %d/%d/%d, 20/50/10000 attendues au contrat", commandes.DefaultPageSize, commandes.MaxPageSize, commandes.MaxPageNumber)
	}
	cas := map[commandes.PageRequest]int{{Number: 1, Size: 20}: 0, {Number: 2, Size: 20}: 20, {Number: 3, Size: 7}: 14, {Number: 10_000, Size: 50}: 499_950}
	for page, attendu := range cas {
		if page.Offset() != attendu {
			t.Errorf("décalage de %+v : %d, %d attendu", page, page.Offset(), attendu)
		}
	}
}

func TestUS032TitreHorsContratDeLaPlaceDeMarcheAucuneCommandeCreee(t *testing.T) {
	for nom, titre := range map[string]string{"titre vide": `""`, "titre de 201 caractères": `"` + strings.Repeat("a", 201) + `"`, "titre absent": ""} {
		t.Run(nom, func(t *testing.T) {
			acheteur, article, _ := nouveauxIdentifiants(t)
			champ := ""
			if titre != "" {
				champ = `"titre":` + titre + `,`
			}
			corps := `{"reservationId":"6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b","articleId":"` + article.String() + `",` + champ + `"echeance":"2026-11-16T10:25:00Z","prix":{"montantCentimes":8000,"devise":"EUR"},"fraisDePort":{"montantCentimes":500,"devise":"EUR"}}`
			serveur := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, corps)
			}))
			t.Cleanup(serveur.Close)
			registre := &registreSimule{parReservation: map[commandes.ReservationID]commandes.Order{}}
			service := commandes.NewService(marche.NewClient(serveur.Client(), serveur.URL), registre, time.Now)
			if _, err := service.Place(t.Context(), acheteur, token.Credential("j"), article); !errors.Is(err, commandes.ErrMarketplaceUnavailable) {
				t.Fatalf("erreur %v, indisponibilité attendue", err)
			}
			if len(registre.enregistrements) != 0 || len(registre.parReservation) != 0 {
				t.Fatalf("commande enregistrée malgré un titre hors contrat")
			}
		})
	}
}
