package logging

import (
	"log/slog"
	"net/http"
	"time"
)

const unknownRoute = "inconnue"

func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		routed := r.WithContext(Resume(r.Context(), r.Header.Get(CorrelationHeader), r.Header.Get(TraceParentHeader)))
		id, _ := CorrelationID(routed.Context())
		w.Header().Set(CorrelationHeader, id)
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
