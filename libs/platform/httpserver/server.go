package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 16 << 10
	shutdownTimeout   = 10 * time.Second
)

func New(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

func Run(ctx context.Context, logger *slog.Logger, servers ...*http.Server) error {
	failures := make(chan error, len(servers))
	for _, server := range servers {
		go func() {
			failures <- listen(ctx, logger, server)
		}()
	}
	var failure error
	select {
	case <-ctx.Done():
	case failure = <-failures:
	}
	return errors.Join(failure, shutdown(logger, servers))
}

func listen(ctx context.Context, logger *slog.Logger, server *http.Server) error {
	logger.InfoContext(ctx, "écoute", slog.String("event", "http.ecoute"), slog.String("adresse", server.Addr))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serveur %s : %w", server.Addr, err)
	}
	return nil
}

func shutdown(logger *slog.Logger, servers []*http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	logger.InfoContext(ctx, "arrêt des serveurs", slog.String("event", "http.arret"))
	var failures []error
	for _, server := range servers {
		if err := server.Shutdown(ctx); err != nil {
			failures = append(failures, fmt.Errorf("arrêt du serveur %s : %w", server.Addr, err))
		}
	}
	return errors.Join(failures...)
}
