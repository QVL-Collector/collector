package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

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
	pageParameter     = "page"
	pageSizeParameter = "taille"
	codeInvalid       = "invalide"
	codeOutOfBounds   = "hors_limites"
)

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
	number, numberErrors := boundedParameter(query, pageParameter, 1, notifications.MaxPageNumber)
	size, sizeErrors := boundedParameter(query, pageSizeParameter, notifications.DefaultPageSize, notifications.MaxPageSize)
	return notifications.PageRequest{Number: number, Size: size}, append(numberErrors, sizeErrors...)
}

func boundedParameter(query url.Values, name string, fallback, maximum int) (int, []problem.FieldError) {
	values, present := query[name]
	if !present {
		return fallback, nil
	}
	if len(values) != 1 || values[0] == "" {
		return 0, []problem.FieldError{{Field: name, Code: codeInvalid}}
	}
	value, err := strconv.Atoi(values[0])
	if err != nil {
		return 0, []problem.FieldError{{Field: name, Code: codeInvalid}}
	}
	if value < 1 || value > maximum {
		return 0, []problem.FieldError{{Field: name, Code: codeOutOfBounds}}
	}
	return value, nil
}
