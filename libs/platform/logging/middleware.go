package logging

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	CorrelationHeader = "X-Correlation-Id"
	unknownRoute      = "inconnue"
)

type correlationContextKey struct{}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationContextKey{}, id)
}

func CorrelationID(ctx context.Context) (string, bool) {
	id, found := ctx.Value(correlationContextKey{}).(string)
	return id, found
}

func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := correlationIDOf(r.Header)
		w.Header().Set(CorrelationHeader, id)
		routed := r.WithContext(WithCorrelationID(r.Context(), id))
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, routed)
		logger.LogAttrs(routed.Context(), slog.LevelInfo, "requête traitée",
			slog.String("event", "http.requete"),
			slog.String("methode", r.Method),
			slog.String("route", routeOf(routed)),
			slog.Int("statut", recorder.status),
			slog.Duration("duree", time.Since(started)),
		)
	})
}

func correlationIDOf(header http.Header) string {
	received, err := uuid.Parse(header.Get(CorrelationHeader))
	if err != nil {
		return uuid.NewString()
	}
	return received.String()
}

func routeOf(r *http.Request) string {
	if r.Pattern == "" {
		return unknownRoute
	}
	return r.Pattern
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if !s.wroteHeader {
		s.status = status
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(body []byte) (int, error) {
	s.wroteHeader = true
	return s.ResponseWriter.Write(body)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
