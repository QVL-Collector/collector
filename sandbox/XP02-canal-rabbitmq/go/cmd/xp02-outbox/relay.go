package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/outbox"
	"collector/sandbox/xp02/internal/report"
)

const defaultRelayInterval = 200 * time.Millisecond

type relaySettings struct {
	amqp           app.AMQP
	labels         report.Labels
	batchSize      int
	interval       time.Duration
	idleTimeout    time.Duration
	confirmTimeout time.Duration
}

func relay(ctx context.Context, log *slog.Logger) error {
	var env config.Env
	cfg := relaySettings{
		amqp:           app.LoadAMQP(&env, binary+"-relay"),
		labels:         app.LoadLabels(&env),
		batchSize:      env.Int("XP02_BATCH", defaultBatch),
		interval:       env.Duration("XP02_RELAY_INTERVAL", defaultRelayInterval),
		idleTimeout:    env.Duration("XP02_IDLE_TIMEOUT", 0),
		confirmTimeout: env.Duration("XP02_CONFIRM_TIMEOUT", defaultConfirmTimeout),
	}
	env.Require(cfg.batchSize > 0, "XP02_BATCH doit être strictement positif")
	env.Require(cfg.interval > 0, "XP02_RELAY_INTERVAL doit être strictement positif")
	env.Require(cfg.confirmTimeout > 0, "XP02_CONFIRM_TIMEOUT doit être strictement positif")
	if err := env.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	pool, err := openPool(ctx, &env)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := outbox.NewStore(pool)
	rep := report.New(os.Stdout, cfg.labels)
	var activity app.Activity
	activity.CancelWhenIdle(ctx, cancel, cfg.idleTimeout)
	relayer := outbox.NewRelay(outbox.RelayConfig{
		Store:          store,
		BatchSize:      cfg.batchSize,
		Interval:       cfg.interval,
		UserID:         cfg.amqp.User,
		ConfirmTimeout: cfg.confirmTimeout,
		Log:            log,
		OnPublished:    func(int) { activity.Touch() },
	})
	runErr := app.Supervisor(cfg.amqp, log, rep).Run(ctx, relayer.Worker())

	rep.Count("relayes_confirmes", relayer.Published(), "messages", "publiés avec confirmation puis marqués")
	rep.Latencies("retard_outbox", relayer.Lags())
	backlogErr := reportBacklog(ctx, store, rep)
	app.ReportFault(rep, runErr)
	return errors.Join(runErr, backlogErr, rep.Err())
}

func reportBacklog(ctx context.Context, store *outbox.Store, rep *report.Reporter) error {
	detached, cancel := app.Detached(ctx)
	defer cancel()
	backlog, err := store.Backlog(detached)
	if err != nil {
		return err
	}
	rep.Count("en_attente", backlog.Count, "messages", "lignes non publiées")
	rep.Seconds("plus_ancienne_en_attente", backlog.Oldest, "")
	return nil
}
