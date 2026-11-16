package reservations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"collector/services/place-de-marche/internal/annonces"
)

const (
	Scope     = "reserver"
	BuyerRole = "acheteur"
)

var (
	ErrArticleNotReservable     = errors.New("article inexistant, non publié ou mis en vente par l'acheteur")
	ErrArticleAlreadyReserved   = errors.New("article déjà réservé par un autre acheteur")
	ErrBuyerAlreadyHoldsAnother = errors.New("l'acheteur a déjà une réservation active sur un autre article")
)

type ID uuid.UUID

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

type Term struct {
	Requested time.Duration
	Ceiling   time.Duration
}

func (t Term) Duration() time.Duration {
	return min(t.Requested, t.Ceiling)
}

type Request struct {
	Article  annonces.ArticleID
	Buyer    BuyerID
	Duration time.Duration
}

type Reservation struct {
	ID          ID
	Article     annonces.ArticleID
	Buyer       BuyerID
	ExpiresAt   time.Time
	Price       annonces.Money
	ShippingFee annonces.Money
}

type Outcome struct {
	Reservation Reservation
	Created     bool
}

type Store interface {
	Reserve(ctx context.Context, request Request) (Outcome, error)
}

type Service struct {
	store Store
	term  Term
}

func NewService(store Store, term Term) *Service {
	return &Service{store: store, term: term}
}

func (s *Service) Reserve(ctx context.Context, article annonces.ArticleID, buyer BuyerID) (Outcome, error) {
	return s.store.Reserve(ctx, Request{Article: article, Buyer: buyer, Duration: s.term.Duration()})
}
