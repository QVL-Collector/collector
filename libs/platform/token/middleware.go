package token

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"collector/libs/platform/problem"
)

const (
	authorizationHeader = "Authorization"
	authenticateHeader  = "WWW-Authenticate"
	authorizationScheme = "Bearer"
)

type claimsContextKey struct{}

func ClaimsFrom(ctx context.Context) (Claims, bool) {
	claims, found := ctx.Value(claimsContextKey{}).(Claims)
	return claims, found
}

func (v *Verifier) Authenticate(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, found := accessTokenOf(r.Header)
		if !found {
			Unauthorized(w)
			return
		}
		claims, err := v.Verify(r.Context(), raw)
		if err != nil {
			logger.InfoContext(r.Context(), "jeton refusé",
				slog.String("event", "auth.jeton_refuse"),
				slog.Any("cause", err),
			)
			Unauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsContextKey{}, claims)))
	})
}

func RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, found := ClaimsFrom(r.Context())
		if !found {
			Unauthorized(w)
			return
		}
		if !claims.HasRole(role) {
			problem.Write(w, problem.Forbidden())
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Unauthorized(w http.ResponseWriter) {
	w.Header().Set(authenticateHeader, authorizationScheme)
	problem.Write(w, problem.Unauthorized())
}

func accessTokenOf(header http.Header) (string, bool) {
	values := header.Values(authorizationHeader)
	if len(values) != 1 {
		return "", false
	}
	scheme, raw, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, authorizationScheme) || raw == "" || strings.ContainsAny(raw, " \t") {
		return "", false
	}
	return raw, true
}
