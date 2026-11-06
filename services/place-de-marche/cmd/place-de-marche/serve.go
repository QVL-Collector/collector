package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"collector/contracts"
	"collector/libs/platform/broker"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/libs/platform/outbox"
	"collector/libs/platform/ratelimit"
	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/api"
	"collector/services/place-de-marche/internal/annonces/controle"
	"collector/services/place-de-marche/internal/annonces/diffusion"
	"collector/services/place-de-marche/internal/annonces/stockage"
	comptesstockage "collector/services/place-de-marche/internal/comptes/stockage"
	"collector/services/place-de-marche/internal/config"
)

const (
	relayConnection   = serviceName + "/outbox"
	resultsConnection = serviceName + "/resultats-controle"
)

var productionSafeguardTiming = controle.SafeguardTiming{
	FirstDelay: 2 * time.Minute,
	MaxDelay:   time.Hour,
	SweepEvery: 30 * time.Second,
}

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
	validator, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		return err
	}
	store := stockage.NewStore(pool, controle.NewRequestEncoder(validator), diffusion.NewDecisionEncoder(validator), comptesstockage.NewStore(pool))
	service := annonces.NewService(store, store)
	guard := ratelimit.NewGuard(cfg.RateLimit, logger, time.Now)
	safeguard, err := controle.NewSafeguard(store, productionSafeguardTiming, logger)
	if err != nil {
		return err
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return httpserver.Run(groupCtx, logger,
			httpserver.New(cfg.APIAddress, publicHandler(logger, service, verifier, guard)),
			httpserver.New(cfg.ManagementAddress, httpserver.ManagementHandler(logger, pool.Ping)),
		)
	})
	group.Go(func() error {
		return relayOutbox(groupCtx, logger, cfg.Channel, pool)
	})
	group.Go(func() error {
		return consumeResults(groupCtx, logger, cfg.Channel, controle.NewResultHandler(validator, service, logger))
	})
	group.Go(func() error {
		return safeguard.Run(groupCtx)
	})
	group.Go(func() error {
		return outbox.NewPurger(pool, logger).Run(groupCtx)
	})
	return group.Wait()
}

func publicHandler(logger *slog.Logger, service api.ArticleService, authenticator api.Authenticator, limiter api.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, service, authenticator, limiter, logger)
	return logging.Middleware(logger, httpserver.Recover(logger, httpserver.SecurityHeaders(mux)))
}

func relayOutbox(ctx context.Context, logger *slog.Logger, channel config.Channel, pool *pgxpool.Pool) error {
	relay := outbox.NewRelay(pool, channel.Endpoint.User, logger)
	return broker.NewSupervisor(channel.Endpoint.Named(relayConnection), logger).Run(ctx, relay.Worker())
}

func consumeResults(ctx context.Context, logger *slog.Logger, channel config.Channel, results *controle.ResultHandler) error {
	consumer := broker.Consumer{
		Queue:             controle.ResultQueue,
		ExpectedPublisher: channel.ResultPublisher,
		Prefetch:          broker.DefaultPrefetch,
		Backoff:           broker.DefaultBackoff,
		Logger:            logger,
	}
	return broker.NewSupervisor(channel.Endpoint.Named(resultsConnection), logger).Run(ctx, consumer.Worker(results.Handle))
}
