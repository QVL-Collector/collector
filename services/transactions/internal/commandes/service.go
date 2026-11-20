package commandes

import (
	"context"
	"fmt"
	"time"

	"collector/libs/platform/token"
	"collector/services/transactions/internal/montants"
)

type Marketplace interface {
	Reserve(ctx context.Context, article ArticleID, buyer token.Credential) (Reservation, error)
}

type Orders interface {
	Record(ctx context.Context, placement Placement) (Recorded, error)
	FindOwned(ctx context.Context, buyer BuyerID, id ID) (Order, error)
	ListOwned(ctx context.Context, buyer BuyerID, page PageRequest) (OrderPage, error)
}

type Clock func() time.Time

type Service struct {
	marketplace Marketplace
	orders      Orders
	now         Clock
}

func NewService(marketplace Marketplace, orders Orders, now Clock) *Service {
	return &Service{marketplace: marketplace, orders: orders, now: now}
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
	recorded, err := s.orders.Record(ctx, Placement{Buyer: buyer, Reservation: reservation, Amounts: amounts})
	if err != nil {
		return Recorded{}, err
	}
	recorded.Order = recorded.Order.AsOf(s.now())
	return recorded, nil
}

func (s *Service) OwnOrder(ctx context.Context, buyer BuyerID, id ID) (Order, error) {
	order, err := s.orders.FindOwned(ctx, buyer, id)
	if err != nil {
		return Order{}, err
	}
	return order.AsOf(s.now()), nil
}

func (s *Service) OwnOrders(ctx context.Context, buyer BuyerID, page PageRequest) (OrderPage, error) {
	orders, err := s.orders.ListOwned(ctx, buyer, page)
	if err != nil {
		return OrderPage{}, err
	}
	return orders.AsOf(s.now()), nil
}
