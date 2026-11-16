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

type credentialContextKey struct{}

func ClaimsFrom(ctx context.Context) (Claims, bool) {
	claims, found := ctx.Value(claimsContextKey{}).(Claims)
	return claims, found
}

func CredentialFrom(ctx context.Context) (Credential, bool) {
	credential, found := ctx.Value(credentialContextKey{}).(Credential)
	return credential, found
}

func withClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
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
			logRefusal(r.Context(), logger, err)
			Unauthorized(w)
			return
		}
		ctx := context.WithValue(withClaims(r.Context(), claims), credentialContextKey{}, Credential(raw))
		next.ServeHTTP(w, r.WithContext(ctx))
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

func logRefusal(ctx context.Context, logger *slog.Logger, cause error) {
	logger.InfoContext(ctx, "jeton refusé",
		slog.String("event", "auth.jeton_refuse"),
		slog.Any("cause", cause),
	)
}

func accessTokenOf(header http.Header) (string, bool) {
	values := header.Values(authorizationHeader)
	if len(values) != 1 {
		return "", false
	}
	scheme, raw, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, authorizationScheme) || !wellFormedToken(raw) {
		return "", false
	}
	return raw, true
}

func wellFormedToken(raw string) bool {
	return raw != "" && !strings.ContainsAny(raw, " \t")
}
