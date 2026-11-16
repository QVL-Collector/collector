package commandes

import (
	"context"
	"fmt"

	"collector/libs/platform/token"
	"collector/services/transactions/internal/montants"
)

type Marketplace interface {
	Reserve(ctx context.Context, article ArticleID, buyer token.Credential) (Reservation, error)
}

type Orders interface {
	Record(ctx context.Context, placement Placement) (Recorded, error)
	FindOwned(ctx context.Context, buyer BuyerID, id ID) (Order, error)
}

type Service struct {
	marketplace Marketplace
	orders      Orders
}

func NewService(marketplace Marketplace, orders Orders) *Service {
	return &Service{marketplace: marketplace, orders: orders}
}

func (s *Service) Place(ctx context.Context, buyer BuyerID, credential token.Credential, article ArticleID) (Recorded, error) {
	reservation, err := s.marketplace.Reserve(ctx, article, credential)
	if err != nil {
		return Recorded{}, err
	}
	if reservation.Article != article {
		return Recorded{}, fmt.Errorf("réservation reçue pour l'article %s au lieu de %s", reservation.Article, article)
	}
	amounts, err := montants.Compute(reservation.Price, reservation.ShippingFee)
	if err != nil {
		return Recorded{}, fmt.Errorf("%w : montants de la réservation %s inexploitables : %w", ErrMarketplaceUnavailable, reservation.ID, err)
	}
	return s.orders.Record(ctx, Placement{Buyer: buyer, Reservation: reservation, Amounts: amounts})
}

func (s *Service) OwnOrder(ctx context.Context, buyer BuyerID, id ID) (Order, error) {
	return s.orders.FindOwned(ctx, buyer, id)
}
