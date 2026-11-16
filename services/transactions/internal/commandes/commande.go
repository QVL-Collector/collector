package commandes

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"collector/services/transactions/internal/montants"
)

var (
	ErrOrderNotFound            = errors.New("commande introuvable")
	ErrArticleNotReservable     = errors.New("article inexistant, non publié ou mis en vente par l'acheteur")
	ErrArticleAlreadyReserved   = errors.New("article déjà réservé par un autre acheteur")
	ErrBuyerAlreadyHoldsAnother = errors.New("l'acheteur a déjà une réservation active sur un autre article")
	ErrMarketplaceUnavailable   = errors.New("place de marché indisponible")
)

type ID uuid.UUID

func ParseID(raw string) (ID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return ID{}, fmt.Errorf("identifiant de commande invalide : %w", err)
	}
	return ID(parsed), nil
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

type BuyerID uuid.UUID

func ParseBuyerID(subject string) (BuyerID, error) {
	parsed, err := uuid.Parse(subject)
	if err != nil {
		return BuyerID{}, fmt.Errorf("identifiant d'acheteur invalide : %w", err)
	}
	return BuyerID(parsed), nil
}

func (id BuyerID) String() string {
	return uuid.UUID(id).String()
}

type ArticleID uuid.UUID

func ParseArticleID(raw string) (ArticleID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return ArticleID{}, fmt.Errorf("identifiant d'article invalide : %w", err)
	}
	return ArticleID(parsed), nil
}

func (id ArticleID) String() string {
	return uuid.UUID(id).String()
}

type ReservationID uuid.UUID

func ParseReservationID(raw string) (ReservationID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return ReservationID{}, fmt.Errorf("identifiant de réservation invalide : %w", err)
	}
	return ReservationID(parsed), nil
}

func (id ReservationID) String() string {
	return uuid.UUID(id).String()
}

type Status string

const (
	StatusAwaitingPayment Status = "en_attente_paiement"
	StatusExpired         Status = "expiree"
)

type Reservation struct {
	ID          ReservationID
	Article     ArticleID
	ExpiresAt   time.Time
	Price       montants.Money
	ShippingFee montants.Money
}

type Placement struct {
	Buyer       BuyerID
	Reservation Reservation
	Amounts     montants.Breakdown
}

type Order struct {
	ID          ID
	Article     ArticleID
	Buyer       BuyerID
	Reservation ReservationID
	Status      Status
	Amounts     montants.Breakdown
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

func (o Order) AsOf(now time.Time) Order {
	if o.Status == StatusAwaitingPayment && !now.Before(o.ExpiresAt) {
		o.Status = StatusExpired
	}
	return o
}

type Recorded struct {
	Order   Order
	Created bool
}
