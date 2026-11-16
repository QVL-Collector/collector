package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/problem"
	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/problemes"
)

const (
	basePath       = "/transactions/v1"
	ordersPath     = basePath + "/commandes"
	buyerRole      = "acheteur"
	maxBodyBytes   = 4 << 10
	articleIDField = "articleId"
	codeRequired   = "obligatoire"
)

var notReservable = problem.Typed(problemes.ArticleNotReservable, "Article non réservable",
	http.StatusUnprocessableEntity, "L'article n'est pas réservable.")

var alreadyReserved = problem.Typed(problemes.ArticleAlreadyReserved, "Article déjà réservé",
	http.StatusConflict, "L'article est réservé par un autre acheteur.")

var activeElsewhere = problem.Typed(problemes.ReservationActive, "Réservation déjà active",
	http.StatusConflict, "Une réservation est déjà en cours sur un autre article.")

type OrderService interface {
	Place(ctx context.Context, buyer commandes.BuyerID, credential token.Credential, article commandes.ArticleID) (commandes.Recorded, error)
	OwnOrder(ctx context.Context, buyer commandes.BuyerID, id commandes.ID) (commandes.Order, error)
}

type Authenticator interface {
	Authenticate(logger *slog.Logger, next http.Handler) http.Handler
}

type RateLimiter interface {
	Limit(next http.Handler) http.Handler
}

type handlers struct {
	orders OrderService
	logger *slog.Logger
}

func Register(mux *http.ServeMux, orders OrderService, authenticator Authenticator, limiter RateLimiter, logger *slog.Logger) {
	h := handlers{orders: orders, logger: logger}
	buyerOnly := func(handler http.Handler) http.Handler {
		return authenticator.Authenticate(logger, token.RequireRole(buyerRole, handler))
	}
	mux.Handle("POST "+ordersPath, buyerOnly(limiter.Limit(http.HandlerFunc(h.place))))
	mux.Handle("GET "+ordersPath+"/{id}", buyerOnly(http.HandlerFunc(h.ownOrder)))
	mux.HandleFunc("/", httpserver.NotFound)
}

type placementPayload struct {
	ArticleID *string `json:"articleId"`
}

func (p placementPayload) article() (commandes.ArticleID, []problem.FieldError) {
	if p.ArticleID == nil {
		return commandes.ArticleID{}, []problem.FieldError{{Field: articleIDField, Code: codeRequired}}
	}
	article, err := commandes.ParseArticleID(*p.ArticleID)
	if err != nil {
		return commandes.ArticleID{}, []problem.FieldError{{Field: articleIDField, Code: problem.CodeInvalid}}
	}
	return article, nil
}

func (h handlers) place(w http.ResponseWriter, r *http.Request) {
	buyer, credential, authenticated := buyerOf(w, r)
	if !authenticated {
		return
	}
	var payload placementPayload
	if !h.decode(w, r, &payload) {
		return
	}
	article, fieldErrors := payload.article()
	if len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	recorded, err := h.orders.Place(r.Context(), buyer, credential, article)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.logPlacement(r.Context(), recorded)
	if !recorded.Created {
		httpjson.Write(w, http.StatusOK, toOrderResponse(recorded.Order))
		return
	}
	w.Header().Set("Location", ordersPath+"/"+recorded.Order.ID.String())
	httpjson.Write(w, http.StatusCreated, toOrderResponse(recorded.Order))
}

func (h handlers) ownOrder(w http.ResponseWriter, r *http.Request) {
	buyer, _, authenticated := buyerOf(w, r)
	if !authenticated {
		return
	}
	id, err := commandes.ParseID(r.PathValue("id"))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return
	}
	order, err := h.orders.OwnOrder(r.Context(), buyer, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toOrderResponse(order))
}

func buyerOf(w http.ResponseWriter, r *http.Request) (commandes.BuyerID, token.Credential, bool) {
	claims, found := token.ClaimsFrom(r.Context())
	credential, presented := token.CredentialFrom(r.Context())
	if !found || !presented {
		token.Unauthorized(w)
		return commandes.BuyerID{}, "", false
	}
	buyer, err := commandes.ParseBuyerID(claims.Subject)
	if err != nil {
		token.Unauthorized(w)
		return commandes.BuyerID{}, "", false
	}
	return buyer, credential, true
}

func (h handlers) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	err := httpjson.DecodeIgnoringUnknownFields(w, r, maxBodyBytes, target)
	if err == nil {
		return true
	}
	var rejection *httpjson.RejectionError
	if errors.As(err, &rejection) {
		problem.Write(w, rejection.Problem())
		return false
	}
	h.writeError(w, r, err)
	return false
}

func (h handlers) logPlacement(ctx context.Context, recorded commandes.Recorded) {
	event := "commandes.commande_creee"
	if !recorded.Created {
		event = "commandes.commande_existante"
	}
	h.logger.InfoContext(ctx, "commande en attente de paiement",
		slog.String("event", event),
		slog.String("commande_id", recorded.Order.ID.String()),
		slog.String("reservation_id", recorded.Order.Reservation.String()),
		slog.String("article_id", recorded.Order.Article.String()),
		slog.String("acheteur_id", recorded.Order.Buyer.String()),
	)
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, commandes.ErrOrderNotFound):
		problem.Write(w, problem.NotFound())
	case errors.Is(err, commandes.ErrArticleNotReservable):
		problem.Write(w, notReservable)
	case errors.Is(err, commandes.ErrArticleAlreadyReserved):
		problem.Write(w, alreadyReserved)
	case errors.Is(err, commandes.ErrBuyerAlreadyHoldsAnother):
		problem.Write(w, activeElsewhere)
	case errors.Is(err, commandes.ErrMarketplaceUnavailable):
		h.logger.WarnContext(r.Context(), "place de marché indisponible", slog.String("event", "commandes.place_de_marche_indisponible"), slog.Any("erreur", err))
		problem.Write(w, problem.Unavailable())
	default:
		h.logger.ErrorContext(r.Context(), "erreur interne", slog.String("event", "commandes.erreur_interne"), slog.Any("erreur", err))
		problem.Write(w, problem.Internal())
	}
}
