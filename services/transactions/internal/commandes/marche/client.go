package marche

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"collector/libs/platform/logging"
	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/montants"
	"collector/services/transactions/internal/problemes"
)

const (
	reservationsPathFormat = "/interne/v1/articles/%s/reservations"
	maxResponseBytes       = 64 << 10
)

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{http: httpClient, baseURL: strings.TrimSuffix(baseURL, "/")}
}

func (c *Client) Reserve(ctx context.Context, article commandes.ArticleID, buyer token.Credential) (commandes.Reservation, error) {
	request, err := c.reservationRequest(ctx, article, buyer)
	if err != nil {
		return commandes.Reservation{}, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return commandes.Reservation{}, fmt.Errorf("%w : %w", commandes.ErrMarketplaceUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return commandes.Reservation{}, fmt.Errorf("%w : lecture de la réponse : %w", commandes.ErrMarketplaceUnavailable, err)
	}
	return reservationOf(response.StatusCode, body)
}

func (c *Client) reservationRequest(ctx context.Context, article commandes.ArticleID, buyer token.Credential) (*http.Request, error) {
	target := c.baseURL + fmt.Sprintf(reservationsPathFormat, article)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("requête de réservation : %w", err)
	}
	request.Header.Set(token.UserTokenHeader, buyer.Reveal())
	request.Header.Set("Accept", "application/json")
	if correlation, found := logging.CorrelationID(ctx); found {
		request.Header.Set(logging.CorrelationHeader, correlation)
	}
	return request, nil
}

func reservationOf(status int, body []byte) (commandes.Reservation, error) {
	switch status {
	case http.StatusOK, http.StatusCreated:
		return decodeReservation(body)
	case http.StatusUnprocessableEntity:
		return commandes.Reservation{}, commandes.ErrArticleNotReservable
	case http.StatusConflict:
		return commandes.Reservation{}, conflictOf(body)
	}
	return commandes.Reservation{}, fmt.Errorf("%w : statut %d", commandes.ErrMarketplaceUnavailable, status)
}

func conflictOf(body []byte) error {
	var refusal struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &refusal); err != nil {
		return fmt.Errorf("%w : refus illisible", commandes.ErrMarketplaceUnavailable)
	}
	switch refusal.Type {
	case problemes.ArticleAlreadyReserved:
		return commandes.ErrArticleAlreadyReserved
	case problemes.ReservationActive:
		return commandes.ErrBuyerAlreadyHoldsAnother
	}
	return fmt.Errorf("%w : refus de type inconnu", commandes.ErrMarketplaceUnavailable)
}

type moneyPayload struct {
	AmountInCents *int64 `json:"montantCentimes"`
	Currency      string `json:"devise"`
}

func (m moneyPayload) money() (montants.Money, error) {
	if m.AmountInCents == nil || montants.Currency(m.Currency) != montants.Euro {
		return montants.Money{}, errors.New("montant incomplet ou devise non prise en charge")
	}
	if *m.AmountInCents < 0 {
		return montants.Money{}, errors.New("montant négatif")
	}
	return montants.Money{Cents: *m.AmountInCents, Currency: montants.Euro}, nil
}

type reservationPayload struct {
	ID          string       `json:"reservationId"`
	ArticleID   string       `json:"articleId"`
	ExpiresAt   time.Time    `json:"echeance"`
	Price       moneyPayload `json:"prix"`
	ShippingFee moneyPayload `json:"fraisDePort"`
}

func decodeReservation(body []byte) (commandes.Reservation, error) {
	var payload reservationPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return commandes.Reservation{}, fmt.Errorf("%w : réservation illisible", commandes.ErrMarketplaceUnavailable)
	}
	reservation, err := payload.reservation()
	if err != nil {
		return commandes.Reservation{}, fmt.Errorf("%w : réservation invalide : %w", commandes.ErrMarketplaceUnavailable, err)
	}
	return reservation, nil
}

func (p reservationPayload) reservation() (commandes.Reservation, error) {
	id, idErr := commandes.ParseReservationID(p.ID)
	article, articleErr := commandes.ParseArticleID(p.ArticleID)
	price, priceErr := p.Price.money()
	shippingFee, shippingErr := p.ShippingFee.money()
	if err := errors.Join(idErr, articleErr, priceErr, shippingErr, expiryError(p.ExpiresAt)); err != nil {
		return commandes.Reservation{}, err
	}
	return commandes.Reservation{ID: id, Article: article, ExpiresAt: p.ExpiresAt, Price: price, ShippingFee: shippingFee}, nil
}

func expiryError(expiresAt time.Time) error {
	if expiresAt.IsZero() {
		return errors.New("échéance absente")
	}
	return nil
}
