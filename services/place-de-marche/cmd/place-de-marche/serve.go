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
	"collector/libs/platform/database"
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
	"collector/services/place-de-marche/internal/reservations"
	reservationsapi "collector/services/place-de-marche/internal/reservations/api"
	reservationsstockage "collector/services/place-de-marche/internal/reservations/stockage"
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
	pool, err := database.Connect(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	verifier, err := token.NewVerifier(ctx, cfg.Token)
	if err != nil {
		return err
	}
	delegation, err := reservationDelegation(verifier, cfg.Reservation)
	if err != nil {
		return err
	}
	validator, err := message.NewValidator(contracts.Messages, message.DefaultMaxSize)
	if err != nil {
		return err
	}
	store := stockage.NewStore(pool, controle.NewRequestEncoder(validator), diffusion.NewDecisionEncoder(validator), comptesstockage.NewStore(pool))
	service := annonces.NewService(store, store)
	reservationService := reservations.NewService(reservationsstockage.NewStore(pool), cfg.Reservation.Term)
	guard := ratelimit.NewGuard(cfg.RateLimit, logger, time.Now)
	safeguard, err := controle.NewSafeguard(store, productionSafeguardTiming, logger)
	if err != nil {
		return err
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return httpserver.Run(groupCtx, logger,
			httpserver.New(cfg.APIAddress, publicHandler(logger, service, verifier, guard)),
			httpserver.New(cfg.InternalAddress, internalHandler(logger, reservationService, delegation)),
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

func internalHandler(logger *slog.Logger, reserver reservationsapi.Reserver, authenticator reservationsapi.Authenticator) http.Handler {
	mux := http.NewServeMux()
	reservationsapi.Register(mux, reserver, authenticator, logger)
	return logging.Middleware(logger, httpserver.Recover(logger, httpserver.SecurityHeaders(mux)))
}

func reservationDelegation(users *token.Verifier, reservation config.Reservation) (token.Delegation, error) {
	service, err := users.ForService(reservation.Caller, reservations.Scope)
	if err != nil {
		return token.Delegation{}, err
	}
	return token.Delegation{Service: service, User: users, UserRole: reservations.BuyerRole}, nil
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
