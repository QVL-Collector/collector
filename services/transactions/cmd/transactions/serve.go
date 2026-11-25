package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"collector/libs/platform/database"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/metrics"
	"collector/libs/platform/ratelimit"
	"collector/libs/platform/serviceclient"
	"collector/libs/platform/token"
	"collector/services/transactions/internal/commandes"
	"collector/services/transactions/internal/commandes/api"
	"collector/services/transactions/internal/commandes/marche"
	"collector/services/transactions/internal/commandes/stockage"
	"collector/services/transactions/internal/config"
)

const marketplaceTimeout = 5 * time.Second

func serve(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.LoadServer(os.Getenv)
	if err != nil {
		return err
	}
	metricsHandler, err := metrics.Install(serviceName, logger)
	if err != nil {
		return err
	}
	pool, err := database.Connect(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	verifier, err := token.NewVerifier(ctx, cfg.Token)
	if err != nil {
		return err
	}
	marketplace := marche.NewClient(serviceclient.New(ctx, cfg.ServiceCredentials, marketplaceTimeout), cfg.MarketplaceURL)
	service := commandes.NewService(marketplace, stockage.NewStore(pool), time.Now)
	guard := ratelimit.NewGuard(cfg.RateLimit, logger, time.Now)
	return httpserver.Run(ctx, logger,
		httpserver.New(cfg.APIAddress, publicHandler(logger, service, verifier, guard)),
		httpserver.New(cfg.ManagementAddress, httpserver.ManagementHandler(logger, pool.Ping)),
		httpserver.New(cfg.MetricsAddress, metricsHandler),
	)
}

func publicHandler(logger *slog.Logger, service api.OrderService, authenticator api.Authenticator, limiter api.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, service, authenticator, limiter, logger)
	return logging.Middleware(logger, metrics.Middleware(httpserver.Recover(logger, httpserver.SecurityHeaders(mux))))
}
