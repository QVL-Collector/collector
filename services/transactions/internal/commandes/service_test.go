package commandes_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/montants"
)

type placeDeMarcheSimulee struct {
	reservations map[commandes.ArticleID]commandes.Reservation
	erreur       error
	jetons       []token.Credential
}

func (p *placeDeMarcheSimulee) Reserve(_ context.Context, article commandes.ArticleID, acheteur token.Credential) (commandes.Reservation, error) {
	p.jetons = append(p.jetons, acheteur)
	if p.erreur != nil {
		return commandes.Reservation{}, p.erreur
	}
	return p.reservations[article], nil
}

type registreSimule struct {
	mu              sync.Mutex
	parReservation  map[commandes.ReservationID]commandes.Order
	enregistrements []commandes.Placement
	listes          []listeDemandee
}

func (r *registreSimule) Record(_ context.Context, placement commandes.Placement) (commandes.Recorded, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enregistrements = append(r.enregistrements, placement)
	if existante, trouvee := r.parReservation[placement.Reservation.ID]; trouvee {
		return commandes.Recorded{Order: existante, Created: false}, nil
	}
	id, _ := commandes.ParseID(uuid.NewString())
	commande := commandes.Order{
		ID: id, Article: placement.Reservation.Article, Title: placement.Reservation.Title, Buyer: placement.Buyer, Reservation: placement.Reservation.ID,
		Status: commandes.StatusAwaitingPayment, Amounts: placement.Amounts, ExpiresAt: placement.Reservation.ExpiresAt, CreatedAt: time.Now(),
	}
	r.parReservation[placement.Reservation.ID] = commande
	return commandes.Recorded{Order: commande, Created: true}, nil
}

func (r *registreSimule) FindOwned(_ context.Context, acheteur commandes.BuyerID, id commandes.ID) (commandes.Order, error) {
	for _, c := range r.parReservation {
		if c.ID == id && c.Buyer == acheteur {
			return c, nil
		}
	}
	return commandes.Order{}, commandes.ErrOrderNotFound
}

func (r *registreSimule) ListOwned(_ context.Context, acheteur commandes.BuyerID, page commandes.PageRequest) (commandes.OrderPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listes = append(r.listes, listeDemandee{acheteur: acheteur, page: page})
	var siennes []commandes.Order
	for _, c := range r.parReservation {
		if c.Buyer == acheteur {
			siennes = append(siennes, c)
		}
	}
	slices.SortFunc(siennes, plusRecenteDAbord)
	debut := min(page.Offset(), len(siennes))
	fin := min(debut+page.Size, len(siennes))
	return commandes.OrderPage{Orders: siennes[debut:fin], Total: len(siennes)}, nil
}

func plusRecenteDAbord(a, b commandes.Order) int {
	if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
		return c
	}
	return bytes.Compare(b.ID[:], a.ID[:])
}

type listeDemandee struct {
	acheteur commandes.BuyerID
	page     commandes.PageRequest
}

type bancDuService struct {
	marche   *placeDeMarcheSimulee
	registre *registreSimule
	service  *commandes.Service
}

func nouveauBancDuService() *bancDuService {
	b := &bancDuService{
		marche:   &placeDeMarcheSimulee{reservations: map[commandes.ArticleID]commandes.Reservation{}},
		registre: &registreSimule{parReservation: map[commandes.ReservationID]commandes.Order{}},
	}
	b.service = commandes.NewService(b.marche, b.registre, func() time.Time { return time.Date(2026, 11, 16, 10, 15, 0, 0, time.UTC) })
	return b
}

func euros(centimes int64) montants.Money {
	return montants.Money{Cents: centimes, Currency: montants.Euro}
}

func nouveauxIdentifiants(t *testing.T) (commandes.BuyerID, commandes.ArticleID, commandes.ReservationID) {
	t.Helper()
	acheteur, err := commandes.ParseBuyerID(uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	article, err := commandes.ParseArticleID(uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := commandes.ParseReservationID(uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	return acheteur, article, reservation
}

func (b *bancDuService) reserverA(article commandes.ArticleID, reservation commandes.ReservationID, prix, frais int64) commandes.Reservation {
	r := commandes.Reservation{ID: reservation, Article: article, ExpiresAt: time.Date(2026, 11, 16, 10, 25, 0, 0, time.UTC), Price: euros(prix), ShippingFee: euros(frais)}
	b.marche.reservations[article] = r
	return r
}

func TestCommandeCreeeAvecLesMontantsDeLaReservation(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, article, reservation := nouveauxIdentifiants(t)
	b.reserverA(article, reservation, 12000, 590)
	enregistree, err := b.service.Place(t.Context(), acheteur, token.Credential("jeton-acheteur"), article)
	if err != nil || !enregistree.Created {
		t.Fatalf("commande non créée : %+v, %v", enregistree, err)
	}
	commande := enregistree.Order
	if commande.Buyer != acheteur || commande.Article != article || commande.Reservation != reservation || commande.Status != commandes.StatusAwaitingPayment {
		t.Fatalf("commande inattendue : %+v", commande)
	}
	attendu := montants.Breakdown{Price: euros(12000), ShippingFee: euros(590), TotalPaid: euros(12590), Commission: euros(630), SellerAmount: euros(11960)}
	if commande.Amounts != attendu {
		t.Fatalf("montants %+v, %+v attendus", commande.Amounts, attendu)
	}
	if len(b.marche.jetons) != 1 || b.marche.jetons[0].Reveal() != "jeton-acheteur" {
		t.Fatalf("jeton de l'acheteur non transmis à la Place de marché")
	}
}

func TestIdempotenceParReservation(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, article, reservation := nouveauxIdentifiants(t)
	b.reserverA(article, reservation, 4500, 600)
	premiere, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if err != nil || !premiere.Created {
		t.Fatalf("première commande : %+v, %v", premiere, err)
	}
	for essai := 2; essai <= 3; essai++ {
		nouvelle, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
		if err != nil || nouvelle.Created || nouvelle.Order.ID != premiere.Order.ID || !nouvelle.Order.ExpiresAt.Equal(premiere.Order.ExpiresAt) {
			t.Fatalf("essai %d : %+v, %v ; même commande non créée attendue", essai, nouvelle, err)
		}
	}
	for _, p := range b.registre.enregistrements {
		if p.Reservation.ID != reservation {
			t.Fatalf("enregistrement sur la réservation %v, %v attendue", p.Reservation.ID, reservation)
		}
	}
	if len(b.registre.parReservation) != 1 {
		t.Fatalf("%d commandes, 1 attendue", len(b.registre.parReservation))
	}
}

func TestDeuxReservationsDeuxCommandes(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, article, reservation := nouveauxIdentifiants(t)
	_, autreArticle, autreReservation := nouveauxIdentifiants(t)
	b.reserverA(article, reservation, 1000, 0)
	b.reserverA(autreArticle, autreReservation, 2000, 0)
	premiere, _ := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	seconde, _ := b.service.Place(t.Context(), acheteur, token.Credential("j"), autreArticle)
	if premiere.Order.ID == seconde.Order.ID || !seconde.Created {
		t.Fatalf("deux réservations ont donné la même commande")
	}
}

func TestReservationDUnAutreArticleRefuseeSansEnregistrement(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, article, reservation := nouveauxIdentifiants(t)
	_, autreArticle, _ := nouveauxIdentifiants(t)
	r := b.reserverA(autreArticle, reservation, 1000, 0)
	b.marche.reservations[article] = r
	if _, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article); err == nil {
		t.Fatalf("réservation d'un autre article acceptée")
	}
	if len(b.registre.enregistrements) != 0 {
		t.Fatalf("commande enregistrée pour une réservation incohérente")
	}
}

func TestErreursDeLaPlaceDeMarchePropageesSansEnregistrement(t *testing.T) {
	for _, attendue := range []error{commandes.ErrArticleNotReservable, commandes.ErrArticleAlreadyReserved, commandes.ErrBuyerAlreadyHoldsAnother, commandes.ErrMarketplaceUnavailable} {
		t.Run(attendue.Error(), func(t *testing.T) {
			b := nouveauBancDuService()
			acheteur, article, _ := nouveauxIdentifiants(t)
			b.marche.erreur = attendue
			if _, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article); !errors.Is(err, attendue) {
				t.Fatalf("erreur %v, %v attendue", err, attendue)
			}
			if len(b.registre.enregistrements) != 0 {
				t.Fatalf("commande enregistrée malgré le refus")
			}
		})
	}
}

func TestMontantsIncoherentsDeLaPlaceDeMarcheSansEnregistrement(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, article, reservation := nouveauxIdentifiants(t)
	b.reserverA(article, reservation, -1, 0)
	_, err := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if err == nil {
		t.Fatalf("prix négatif accepté")
	}
	if len(b.registre.enregistrements) != 0 {
		t.Fatalf("commande enregistrée avec un prix négatif")
	}
	t.Logf("prix négatif renvoyé par la Place de marché : %v (indisponibilité : %t)", err, errors.Is(err, commandes.ErrMarketplaceUnavailable))
}

func TestCommandePropreSeulement(t *testing.T) {
	b := nouveauBancDuService()
	acheteur, article, reservation := nouveauxIdentifiants(t)
	autre, _, _ := nouveauxIdentifiants(t)
	b.reserverA(article, reservation, 1000, 0)
	enregistree, _ := b.service.Place(t.Context(), acheteur, token.Credential("j"), article)
	if lue, err := b.service.OwnOrder(t.Context(), acheteur, enregistree.Order.ID); err != nil || lue.ID != enregistree.Order.ID {
		t.Fatalf("lecture par le propriétaire : %v", err)
	}
	if _, err := b.service.OwnOrder(t.Context(), autre, enregistree.Order.ID); !errors.Is(err, commandes.ErrOrderNotFound) {
		t.Fatalf("lecture par un autre acheteur : %v, introuvable attendu", err)
	}
}
