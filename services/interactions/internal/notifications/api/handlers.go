package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/problem"
	"collector/libs/platform/token"
	"collector/services/interactions/internal/notifications"
)

const (
	basePath          = "/interactions/v1"
	notificationsPath = basePath + "/notifications"
	sellerRole        = "vendeur"
)

var pageLimits = httpserver.PageLimits{
	DefaultSize: notifications.DefaultPageSize,
	MaxSize:     notifications.MaxPageSize,
	MaxNumber:   notifications.MaxPageNumber,
}

type NotificationLister interface {
	ListForSeller(ctx context.Context, seller notifications.SellerID, page notifications.PageRequest) (notifications.Page, error)
}

type Authenticator interface {
	Authenticate(logger *slog.Logger, next http.Handler) http.Handler
}

type handlers struct {
	notifications NotificationLister
	logger        *slog.Logger
}

func Register(mux *http.ServeMux, lister NotificationLister, authenticator Authenticator, logger *slog.Logger) {
	h := handlers{notifications: lister, logger: logger}
	sellerOnly := func(handler http.HandlerFunc) http.Handler {
		return authenticator.Authenticate(logger, token.RequireRole(sellerRole, handler))
	}
	mux.Handle("GET "+notificationsPath, sellerOnly(h.listOwnNotifications))
	mux.HandleFunc("/", httpserver.NotFound)
}

func (h handlers) listOwnNotifications(w http.ResponseWriter, r *http.Request) {
	seller, authenticated := sellerOf(w, r)
	if !authenticated {
		return
	}
	request, fieldErrors := pageRequestOf(r.URL.Query())
	if len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	page, err := h.notifications.ListForSeller(r.Context(), seller, request)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "erreur interne", slog.String("event", "notifications.erreur_interne"), slog.Any("erreur", err))
		problem.Write(w, problem.Internal())
		return
	}
	httpjson.Write(w, http.StatusOK, toNotificationPageResponse(page, request))
}

func sellerOf(w http.ResponseWriter, r *http.Request) (notifications.SellerID, bool) {
	claims, found := token.ClaimsFrom(r.Context())
	if !found {
		token.Unauthorized(w)
		return notifications.SellerID{}, false
	}
	seller, err := notifications.ParseSellerID(claims.Subject)
	if err != nil {
		token.Unauthorized(w)
		return notifications.SellerID{}, false
	}
	return seller, true
}

func pageRequestOf(query url.Values) (notifications.PageRequest, []problem.FieldError) {
	pagination, fieldErrors := httpserver.PaginationOf(query, pageLimits)
	return notifications.PageRequest(pagination), fieldErrors
}
