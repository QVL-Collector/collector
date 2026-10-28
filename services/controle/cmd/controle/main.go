package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"collector/contracts"
	"collector/libs/platform/broker"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/services/controle/internal/canal"
	"collector/services/controle/internal/config"
	"collector/services/controle/internal/regles"
)

const (
	serviceName        = "controle"
	requestsConnection = serviceName + "/demandes"
	envLogLevel        = "JOURNAL_NIVEAU"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	level, levelErr := logging.ParseLevel(os.Getenv(envLogLevel))
	logger := logging.New(serviceName, os.Stdout, level)
	if levelErr != nil {
		logger.WarnContext(ctx, "niveau de journal ignoré", slog.String("event", "demarrage.configuration"), slog.Any("erreur", levelErr))
	}
	if err := serve(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "arrêt sur erreur", slog.String("event", "arret.erreur"), slog.Any("erreur", err))
		return 1
	}
	return 0
}

func serve(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.LoadServer(os.Getenv)
	if err != nil {
		return err
	}
	validator, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		return err
	}
	responder := canal.NewResponder(validator, regles.Reference(), logger)
	consumer := broker.Consumer{
		Queue:             canal.RequestQueue,
		ExpectedPublisher: cfg.RequestPublisher,
		Prefetch:          broker.DefaultPrefetch,
		Backoff:           broker.DefaultBackoff,
		Logger:            logger,
	}
	supervisor := broker.NewSupervisor(cfg.Endpoint.Named(requestsConnection), logger)
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return httpserver.Run(groupCtx, logger, httpserver.New(cfg.ManagementAddress, httpserver.ManagementHandler(logger, supervisor.Ready)))
	})
	group.Go(func() error {
		return supervisor.Run(groupCtx, consumer.WorkerWithPublisher(cfg.Endpoint.User, responder.HandlerFor))
	})
	return group.Wait()
}
