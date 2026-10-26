package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/problem"
)

const readinessTimeout = 2 * time.Second

type ReadinessCheck func(ctx context.Context) error

type healthStatus struct {
	Status string `json:"statut"`
}

func ManagementHandler(logger *slog.Logger, ready ReadinessCheck) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sante", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.Write(w, http.StatusOK, healthStatus{Status: "vivant"})
	})
	mux.HandleFunc("GET /disponibilite", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		if err := ready(ctx); err != nil {
			logger.WarnContext(ctx, "service non disponible",
				slog.String("event", "sante.indisponible"),
				slog.Any("erreur", err),
			)
			problem.Write(w, problem.Unavailable())
			return
		}
		httpjson.Write(w, http.StatusOK, healthStatus{Status: "disponible"})
	})
	mux.HandleFunc("/", NotFound)
	return mux
}
