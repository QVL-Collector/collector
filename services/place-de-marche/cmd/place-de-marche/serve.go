package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/api"
	"collector/services/place-de-marche/internal/annonces/stockage"
	"collector/services/place-de-marche/internal/config"
)

func serve(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.LoadServer(os.Getenv)
	if err != nil {
		return err
	}
	pool, err := stockage.Connect(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	verifier, err := token.NewVerifier(ctx, cfg.Token)
	if err != nil {
		return err
	}
	store := stockage.NewStore(pool)
	service := annonces.NewService(store, store)
	return httpserver.Run(ctx, logger,
		httpserver.New(cfg.APIAddress, publicHandler(logger, service, verifier)),
		httpserver.New(cfg.ManagementAddress, httpserver.ManagementHandler(logger, pool.Ping)),
	)
}

func publicHandler(logger *slog.Logger, service api.ArticleService, authenticator api.Authenticator) http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, service, authenticator, logger)
	return logging.Middleware(logger, httpserver.Recover(logger, httpserver.SecurityHeaders(mux)))
}
