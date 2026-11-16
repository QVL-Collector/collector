package commandes_test

import (
	"sync"
	"testing"
	"time"

	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/montants"
)

var echeanceDeReference = time.Date(2026, 11, 16, 10, 25, 0, 0, time.UTC)

type horlogeReglable struct {
	mu         sync.Mutex
	maintenant time.Time
}

func (h *horlogeReglable) lire() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.maintenant
}

func (h *horlogeReglable) regler(instant time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.maintenant = instant
}

func commandeEnAttente() commandes.Order {
	return commandes.Order{Status: commandes.StatusAwaitingPayment, ExpiresAt: echeanceDeReference, Amounts: montantsDeReference()}
}

func montantsDeReference() montants.Breakdown {
	return montants.Breakdown{Price: euros(8000), ShippingFee: euros(500), TotalPaid: euros(8500), Commission: euros(425), SellerAmount: euros(8075)}
}

func TestUS031AC4StatutSelonLEcheance(t *testing.T) {
	cas := []struct {
		nom        string
		maintenant time.Time
		attendu    commandes.Status
	}{
		{"bien avant l'échéance", echeanceDeReference.Add(-10 * time.Minute), commandes.StatusAwaitingPayment},
		{"une nanoseconde avant l'échéance", echeanceDeReference.Add(-time.Nanosecond), commandes.StatusAwaitingPayment},
		{"à l'échéance exacte", echeanceDeReference, commandes.StatusExpired},
		{"une nanoseconde après l'échéance", echeanceDeReference.Add(time.Nanosecond), commandes.StatusExpired},
		{"un an après l'échéance", echeanceDeReference.AddDate(1, 0, 0), commandes.StatusExpired},
		{"échéance exacte exprimée dans un autre fuseau", echeanceDeReference.In(time.FixedZone("Paris", 3600)), commandes.StatusExpired},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			commande := commandeEnAttente()
			lue := commande.AsOf(c.maintenant)
			if lue.Status != c.attendu {
				t.Fatalf("statut %q, %q attendu", lue.Status, c.attendu)
			}
			if commande.Status != commandes.StatusAwaitingPayment {
				t.Fatalf("la commande d'origine a changé de statut : %q", commande.Status)
			}
			if !lue.ExpiresAt.Equal(echeanceDeReference) || lue.Amounts != commande.Amounts {
				t.Fatalf("échéance ou montants changés par la lecture : %+v", lue)
			}
		})
	}
}

func TestUS031AC4CommandeExpireeJamaisReactivee(t *testing.T) {
	expiree := commandeEnAttente().AsOf(echeanceDeReference)
	for _, instant := range []time.Time{echeanceDeReference.Add(-time.Hour), echeanceDeReference.Add(-time.Nanosecond), echeanceDeReference.Add(time.Hour)} {
		if lue := expiree.AsOf(instant); lue.Status != commandes.StatusExpired {
			t.Fatalf("commande expirée relue à %s : %q", instant, lue.Status)
		}
	}
}

func TestUS031AC4StatutsAuContrat(t *testing.T) {
	if commandes.StatusAwaitingPayment != "en_attente_paiement" || commandes.StatusExpired != "expiree" {
		t.Fatalf("statuts %q et %q, en_attente_paiement et expiree attendus au contrat", commandes.StatusAwaitingPayment, commandes.StatusExpired)
	}
}

func nouveauBancHorloge(t *testing.T, horloge *horlogeReglable) (*bancDuService, commandes.BuyerID, commandes.ArticleID) {
	t.Helper()
	b := &bancDuService{
		marche:   &placeDeMarcheSimulee{reservations: map[commandes.ArticleID]commandes.Reservation{}},
		registre: &registreSimule{parReservation: map[commandes.ReservationID]commandes.Order{}},
	}
	b.service = commandes.NewService(b.marche, b.registre, horloge.lire)
	acheteur, article, _ := nouveauxIdentifiants(t)
	return b, acheteur, article
}

func (b *bancDuService) reserverJusquA(t *testing.T, article commandes.ArticleID, echeance time.Time) commandes.ReservationID {
	t.Helper()
	_, _, reservation := nouveauxIdentifiants(t)
	b.marche.reservations[article] = commandes.Reservation{ID: reservation, Article: article, ExpiresAt: echeance, Price: euros(8000), ShippingFee: euros(500)}
	return reservation
}

func (b *bancDuService) premiereCommande(t *testing.T, acheteur commandes.BuyerID, article commandes.ArticleID) commandes.Recorded {
	t.Helper()
	premiere, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if err != nil || !premiere.Created || premiere.Order.Status != commandes.StatusAwaitingPayment {
		t.Fatalf("première commande : %+v, %v ; commande créée en attente de paiement attendue", premiere, err)
	}
	return premiere
}

func TestUS031AC4CommandeLueExpireeApresLEcheance(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	b.reserverJusquA(t, article, echeanceDeReference)
	creee, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if err != nil || !creee.Created || creee.Order.Status != commandes.StatusAwaitingPayment {
		t.Fatalf("création : %+v, %v", creee, err)
	}
	horloge.regler(echeanceDeReference.Add(-time.Nanosecond))
	if lue, err := b.service.OwnOrder(t.Context(), acheteur, creee.Order.ID); err != nil || lue.Status != commandes.StatusAwaitingPayment {
		t.Fatalf("juste avant l'échéance : %+v, %v", lue, err)
	}
	horloge.regler(echeanceDeReference)
	lue, err := b.service.OwnOrder(t.Context(), acheteur, creee.Order.ID)
	if err != nil || lue.Status != commandes.StatusExpired || lue.ID != creee.Order.ID || !lue.ExpiresAt.Equal(echeanceDeReference) {
		t.Fatalf("à l'échéance : %+v, %v ; même commande expiree attendue", lue, err)
	}
}

func TestUS031AC4NouvelEssaiApresEcheanceNouvelleCommande(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	premiereReservation := b.reserverJusquA(t, article, echeanceDeReference)
	premiere := b.premiereCommande(t, acheteur, article)
	horloge.regler(echeanceDeReference.Add(time.Second))
	nouvelleEcheance := echeanceDeReference.Add(10 * time.Minute)
	nouvelleReservation := b.reserverJusquA(t, article, nouvelleEcheance)
	seconde, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if err != nil || !seconde.Created {
		t.Fatalf("nouvel essai après l'échéance : %+v, %v ; nouvelle commande attendue", seconde, err)
	}
	if seconde.Order.ID == premiere.Order.ID || seconde.Order.Reservation != nouvelleReservation || seconde.Order.Reservation == premiereReservation {
		t.Fatalf("nouvelle commande %+v sur l'ancienne réservation ou l'ancienne commande", seconde.Order)
	}
	if seconde.Order.Status != commandes.StatusAwaitingPayment || !seconde.Order.ExpiresAt.Equal(nouvelleEcheance) {
		t.Fatalf("nouvelle commande : %+v", seconde.Order)
	}
	ancienne, err := b.service.OwnOrder(t.Context(), acheteur, premiere.Order.ID)
	if err != nil || ancienne.Status != commandes.StatusExpired {
		t.Fatalf("ancienne commande : %+v, %v ; lisible et expiree attendue", ancienne, err)
	}
	if len(b.registre.parReservation) != 2 {
		t.Fatalf("%d commandes enregistrées, 2 attendues", len(b.registre.parReservation))
	}
}

func TestUS031AC2RelanceAvantEcheanceMemeCommandeEnAttente(t *testing.T) {
	horloge := &horlogeReglable{maintenant: echeanceDeReference.Add(-5 * time.Minute)}
	b, acheteur, article := nouveauBancHorloge(t, horloge)
	b.reserverJusquA(t, article, echeanceDeReference)
	premiere := b.premiereCommande(t, acheteur, article)
	horloge.regler(echeanceDeReference.Add(-time.Second))
	relance, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if err != nil || relance.Created || relance.Order.ID != premiere.Order.ID || relance.Order.Status != commandes.StatusAwaitingPayment || !relance.Order.ExpiresAt.Equal(premiere.Order.ExpiresAt) {
		t.Fatalf("relance : %+v, %v ; même commande, même échéance, en attente attendue", relance, err)
	}
}
