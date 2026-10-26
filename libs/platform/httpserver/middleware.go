package httpserver

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"collector/libs/platform/problem"
)

var securityHeaders = map[string]string{
	"Cache-Control":           "no-store",
	"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	"Referrer-Policy":         "no-referrer",
	"X-Content-Type-Options":  "nosniff",
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, value := range securityHeaders {
			w.Header().Set(name, value)
		}
		next.ServeHTTP(w, r)
	})
}

func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if err, isError := recovered.(error); isError && errors.Is(err, http.ErrAbortHandler) {
				panic(recovered)
			}
			logger.ErrorContext(r.Context(), "panique interceptée",
				slog.String("event", "http.panique"),
				slog.String("cause", fmt.Sprint(recovered)),
			)
			problem.Write(w, problem.Internal())
		}()
		next.ServeHTTP(w, r)
	})
}

func NotFound(w http.ResponseWriter, _ *http.Request) {
	problem.Write(w, problem.NotFound())
}
