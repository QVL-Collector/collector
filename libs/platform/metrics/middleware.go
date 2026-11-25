package metrics

import (
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	httpScope           = "collector/libs/platform/metrics"
	requestDurationName = "http.server.request.duration"
	methodKey           = "http.request.method"
	routeKey            = "http.route"
	statusCodeKey       = "http.response.status_code"
	unknownRoute        = "inconnue"
	otherMethod         = "_OTHER"
)

var latencyBoundaries = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1, 2.5, 5, 10}

var knownMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodOptions: true,
	http.MethodConnect: true,
	http.MethodTrace:   true,
}

func Middleware(next http.Handler) http.Handler {
	duration := Histogram(otel.Meter(httpScope), requestDurationName, "s",
		"Durée de traitement des requêtes HTTP, par méthode, modèle de route et code de réponse", latencyBoundaries)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		duration.Record(r.Context(), time.Since(started).Seconds(), metric.WithAttributes(
			attribute.String(methodKey, methodOf(r)),
			attribute.String(routeKey, routeOf(r)),
			attribute.Int(statusCodeKey, recorder.status),
		))
	})
}

func methodOf(r *http.Request) string {
	if knownMethods[r.Method] {
		return r.Method
	}
	return otherMethod
}

func routeOf(r *http.Request) string {
	if r.Pattern == "" {
		return unknownRoute
	}
	_, path, hasMethod := strings.Cut(r.Pattern, " ")
	if !hasMethod {
		return r.Pattern
	}
	return strings.TrimSpace(path)
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
