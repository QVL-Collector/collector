package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/sync/errgroup"

	"collector/contracts"
	"collector/libs/platform/broker"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/libs/platform/token"
	"collector/services/interactions/internal/config"
	"collector/services/interactions/internal/notifications/api"
	"collector/services/interactions/internal/notifications/canal"
	"collector/services/interactions/internal/notifications/stockage"
)

const decisionsConnection = serviceName + "/publication-decidee"

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
	store := stockage.NewStore(pool)
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return httpserver.Run(groupCtx, logger,
			httpserver.New(cfg.APIAddress, publicHandler(logger, store, verifier)),
			httpserver.New(cfg.ManagementAddress, httpserver.ManagementHandler(logger, pool.Ping)),
		)
	})
	group.Go(func() error {
		return consumeDecisions(groupCtx, logger, cfg.Channel, canal.NewDecisionHandler(validator, store, logger))
	})
	return group.Wait()
}

func publicHandler(logger *slog.Logger, lister api.NotificationLister, authenticator api.Authenticator) http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, lister, authenticator, logger)
	return logging.Middleware(logger, httpserver.Recover(logger, httpserver.SecurityHeaders(mux)))
}

func consumeDecisions(ctx context.Context, logger *slog.Logger, channel config.Channel, decisions *canal.DecisionHandler) error {
	consumer := broker.Consumer{
		Queue:             canal.DecisionQueue,
		ExpectedPublisher: channel.DecisionPublisher,
		Prefetch:          broker.DefaultPrefetch,
		Backoff:           broker.DefaultBackoff,
		Logger:            logger,
	}
	return broker.NewSupervisor(channel.Endpoint.Named(decisionsConnection), logger).Run(ctx, consumer.Worker(decisions.Handle))
}
