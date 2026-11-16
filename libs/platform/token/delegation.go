package token

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"collector/libs/platform/problem"
)

const UserTokenHeader = "X-Collector-User-Token"

var errMissingUserToken = errors.New("jeton de l'utilisateur absent ou mal formé")

type Delegation struct {
	Service  *Verifier
	User     *Verifier
	UserRole string
}

func (d Delegation) Authenticate(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !d.callerAuthenticated(r, logger) {
			Unauthorized(w)
			return
		}
		user, err := d.propagatedUser(r.Context(), r.Header)
		if err != nil {
			logRefusal(r.Context(), logger, err)
			problem.Write(w, problem.Forbidden())
			return
		}
		next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), user)))
	})
}

func (d Delegation) callerAuthenticated(r *http.Request, logger *slog.Logger) bool {
	raw, found := accessTokenOf(r.Header)
	if !found {
		return false
	}
	if _, err := d.Service.Verify(r.Context(), raw); err != nil {
		logRefusal(r.Context(), logger, err)
		return false
	}
	return true
}

func (d Delegation) propagatedUser(ctx context.Context, header http.Header) (Claims, error) {
	raw, found := userTokenOf(header)
	if !found {
		return Claims{}, errMissingUserToken
	}
	user, err := d.User.Verify(ctx, raw)
	if err != nil {
		return Claims{}, err
	}
	if !user.HasRole(d.UserRole) {
		return Claims{}, errors.New("rôle exigé absent du jeton de l'utilisateur")
	}
	return user, nil
}

func userTokenOf(header http.Header) (string, bool) {
	values := header.Values(UserTokenHeader)
	if len(values) != 1 || !wellFormedToken(values[0]) {
		return "", false
	}
	return values[0], true
}
