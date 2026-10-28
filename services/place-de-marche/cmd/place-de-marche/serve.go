package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"collector/contracts"
	"collector/libs/platform/broker"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/libs/platform/outbox"
	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/annonces"
	"collector/services/place-de-marche/internal/annonces/api"
	"collector/services/place-de-marche/internal/annonces/controle"
	"collector/services/place-de-marche/internal/annonces/stockage"
	"collector/services/place-de-marche/internal/config"
)

const (
	relayConnection   = serviceName + "/outbox"
	resultsConnection = serviceName + "/resultats-controle"
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
	validator, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		return err
	}
	store := stockage.NewStore(pool, controle.NewRequestEncoder(validator))
	service := annonces.NewService(store, store)
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return httpserver.Run(groupCtx, logger,
			httpserver.New(cfg.APIAddress, publicHandler(logger, service, verifier)),
			httpserver.New(cfg.ManagementAddress, httpserver.ManagementHandler(logger, pool.Ping)),
		)
	})
	group.Go(func() error {
		return relayOutbox(groupCtx, logger, cfg.Channel, pool)
	})
	group.Go(func() error {
		return consumeResults(groupCtx, logger, cfg.Channel, controle.NewResultHandler(validator, service, logger))
	})
	return group.Wait()
}

func publicHandler(logger *slog.Logger, service api.ArticleService, authenticator api.Authenticator) http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, service, authenticator, logger)
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
