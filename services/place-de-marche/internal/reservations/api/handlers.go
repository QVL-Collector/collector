package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/problem"
	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/reservations"
)

const reservationsPath = "/interne/v1/articles/{id}/reservations"

var notReservable = problem.Typed("urn:collector:probleme:article-non-reservable", "Article non réservable",
	http.StatusUnprocessableEntity, "L'article n'est pas réservable.")

var alreadyReserved = problem.Typed("urn:collector:probleme:article-deja-reserve", "Article déjà réservé",
	http.StatusConflict, "L'article est réservé par un autre acheteur.")

var activeElsewhere = problem.Typed("urn:collector:probleme:reservation-deja-active", "Réservation déjà active",
	http.StatusConflict, "L'acheteur a déjà une réservation active sur un autre article.")

type Reserver interface {
	Reserve(ctx context.Context, article annonces.ArticleID, buyer reservations.BuyerID) (reservations.Outcome, error)
}

type Authenticator interface {
	Authenticate(logger *slog.Logger, next http.Handler) http.Handler
}

type handlers struct {
	reservations Reserver
	logger       *slog.Logger
}

func Register(mux *http.ServeMux, reserver Reserver, authenticator Authenticator, logger *slog.Logger) {
	h := handlers{reservations: reserver, logger: logger}
	mux.Handle("POST "+reservationsPath, authenticator.Authenticate(logger, http.HandlerFunc(h.reserve)))
	mux.HandleFunc("/", httpserver.NotFound)
}

func (h handlers) reserve(w http.ResponseWriter, r *http.Request) {
	buyer, identified := buyerOf(r)
	if !identified {
		problem.Write(w, problem.Forbidden())
		return
	}
	article, err := annonces.ParseArticleID(r.PathValue("id"))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return
	}
	outcome, err := h.reservations.Reserve(r.Context(), article, buyer)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.logReservation(r.Context(), outcome)
	httpjson.Write(w, statusOf(outcome), toReservationResponse(outcome.Reservation))
}

func buyerOf(r *http.Request) (reservations.BuyerID, bool) {
	claims, found := token.ClaimsFrom(r.Context())
	if !found {
		return reservations.BuyerID{}, false
	}
	buyer, err := reservations.ParseBuyerID(claims.Subject)
	return buyer, err == nil
}

func statusOf(outcome reservations.Outcome) int {
	if outcome.Created {
		return http.StatusCreated
	}
	return http.StatusOK
}

func (h handlers) logReservation(ctx context.Context, outcome reservations.Outcome) {
	h.logger.InfoContext(ctx, "article réservé",
		slog.String("event", "reservations.article_reserve"),
		slog.String("reservation_id", outcome.Reservation.ID.String()),
		slog.String("article_id", outcome.Reservation.Article.String()),
		slog.String("acheteur_id", outcome.Reservation.Buyer.String()),
		slog.Bool("nouvelle", outcome.Created),
	)
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, reservations.ErrArticleNotReservable):
		problem.Write(w, notReservable)
	case errors.Is(err, reservations.ErrArticleAlreadyReserved):
		problem.Write(w, alreadyReserved)
	case errors.Is(err, reservations.ErrBuyerAlreadyHoldsAnother):
		problem.Write(w, activeElsewhere)
	default:
		h.logger.ErrorContext(r.Context(), "erreur interne", slog.String("event", "reservations.erreur_interne"), slog.Any("erreur", err))
		problem.Write(w, problem.Internal())
	}
}

type moneyResponse struct {
	AmountInCents int64  `json:"montantCentimes"`
	Currency      string `json:"devise"`
}

type reservationResponse struct {
	ID          string        `json:"reservationId"`
	ArticleID   string        `json:"articleId"`
	ExpiresAt   time.Time     `json:"echeance"`
	Price       moneyResponse `json:"prix"`
	ShippingFee moneyResponse `json:"fraisDePort"`
}

func toReservationResponse(reservation reservations.Reservation) reservationResponse {
	return reservationResponse{
		ID:          reservation.ID.String(),
		ArticleID:   reservation.Article.String(),
		ExpiresAt:   reservation.ExpiresAt.UTC(),
		Price:       moneyResponseOf(reservation.Price),
		ShippingFee: moneyResponseOf(reservation.ShippingFee),
	}
}

func moneyResponseOf(money annonces.Money) moneyResponse {
	return moneyResponse{AmountInCents: money.Cents, Currency: string(money.Currency)}
}
