package api

import (
	"time"

	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/montants"
)

type moneyResponse struct {
	AmountInCents int64  `json:"montantCentimes"`
	Currency      string `json:"devise"`
}

type orderResponse struct {
	ID           string        `json:"id"`
	ArticleID    string        `json:"articleId"`
	Title        string        `json:"titre"`
	Status       string        `json:"statut"`
	Price        moneyResponse `json:"prix"`
	ShippingFee  moneyResponse `json:"fraisDePort"`
	TotalPaid    moneyResponse `json:"totalPaye"`
	Commission   moneyResponse `json:"commission"`
	SellerAmount moneyResponse `json:"montantVendeur"`
	ExpiresAt    time.Time     `json:"echeance"`
	CreatedAt    time.Time     `json:"creeLe"`
}

func toOrderResponse(order commandes.Order) orderResponse {
	return orderResponse{
		ID:           order.ID.String(),
		ArticleID:    order.Article.String(),
		Title:        order.Title,
		Status:       string(order.Status),
		Price:        moneyResponseOf(order.Amounts.Price),
		ShippingFee:  moneyResponseOf(order.Amounts.ShippingFee),
		TotalPaid:    moneyResponseOf(order.Amounts.TotalPaid),
		Commission:   moneyResponseOf(order.Amounts.Commission),
		SellerAmount: moneyResponseOf(order.Amounts.SellerAmount),
		ExpiresAt:    order.ExpiresAt.UTC(),
		CreatedAt:    order.CreatedAt.UTC(),
	}
}

type orderPageResponse struct {
	Orders []orderResponse `json:"elements"`
	Page   int             `json:"page"`
	Size   int             `json:"taille"`
	Total  int             `json:"total"`
}

func toOrderPageResponse(page commandes.OrderPage, request commandes.PageRequest) orderPageResponse {
	orders := make([]orderResponse, 0, len(page.Orders))
	for _, order := range page.Orders {
		orders = append(orders, toOrderResponse(order))
	}
	return orderPageResponse{Orders: orders, Page: request.Number, Size: request.Size, Total: page.Total}
}

func moneyResponseOf(money montants.Money) moneyResponse {
	return moneyResponse{AmountInCents: money.Cents, Currency: string(money.Currency)}
}
