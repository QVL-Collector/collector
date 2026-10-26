package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/broker"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/ledger"
	"collector/sandbox/xp02/internal/report"
)

const (
	binary         = "xp02-consume"
	ledgerMemory   = "memory"
	ledgerPostgres = "postgres"
	policyReject   = "reject"
	policyRequeue  = "requeue"
)

type settings struct {
	amqp              app.AMQP
	labels            report.Labels
	queue             string
	expectedPublisher string
	prefetch          int
	maxMessageBytes   int
	invalidPolicy     broker.Disposition
	processingDelay   time.Duration
	idleTimeout       time.Duration
	ledger            string
	pgDSN             string
}

type journal interface {
	Record(ctx context.Context, eventID string) (bool, error)
	Total(ctx context.Context) (int64, error)
}

func main() {
	log := app.Logger(binary)
	if err := run(log); err != nil {
		log.Error("arrêt sur erreur", "erreur", err)
		os.Exit(1)
	}
}

func load() (settings, error) {
	var env config.Env
	cfg := settings{
		amqp:              app.LoadAMQP(&env, binary),
		labels:            app.LoadLabels(&env),
		queue:             env.Required("XP02_QUEUE"),
		expectedPublisher: env.Required("XP02_EXPECTED_PUBLISHER"),
		prefetch:          env.Int("XP02_PREFETCH", 10),
		maxMessageBytes:   env.Int("XP02_MAX_MESSAGE_BYTES", 64*1024),
		invalidPolicy:     parsePolicy(&env, env.String("XP02_INVALID_POLICY", policyReject)),
		processingDelay:   env.Duration("XP02_PROCESSING_DELAY", 0),
		idleTimeout:       env.Duration("XP02_IDLE_TIMEOUT", 0),
		ledger:            env.String("XP02_LEDGER", ledgerMemory),
		pgDSN:             env.String("XP02_PG_DSN", ""),
	}
	env.Require(cfg.prefetch > 0, "XP02_PREFETCH doit être strictement positif")
	env.Require(cfg.maxMessageBytes > 0, "XP02_MAX_MESSAGE_BYTES doit être strictement positif")
	env.Require(cfg.ledger == ledgerMemory || cfg.ledger == ledgerPostgres, "XP02_LEDGER attend memory ou postgres")
	return cfg, env.Err()
}

func parsePolicy(env *config.Env, raw string) broker.Disposition {
	switch raw {
	case policyReject:
		return broker.Reject
	case policyRequeue:
		return broker.Requeue
	}
	env.Require(false, "XP02_INVALID_POLICY attend reject ou requeue")
	return broker.Reject
}

func run(log *slog.Logger) error {
	cfg, err := load()
	if err != nil {
		return err
	}
	ctx, stop := app.SignalContext()
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	receptions, closeJournal, err := openJournal(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeJournal()

	rep := report.New(os.Stdout, cfg.labels)
	h := newHandler(cfg, receptions, log)
	h.activity.CancelWhenIdle(ctx, cancel, cfg.idleTimeout)

	consumer := broker.Consumer{Queue: cfg.queue, Prefetch: cfg.prefetch, Backoff: app.DefaultBackoff, Log: log}
	runErr := app.Supervisor(cfg.amqp, log, rep).Run(ctx, consumer.Run(h.handle))

	h.report(ctx, rep)
	app.ReportFault(rep, runErr)
	return errors.Join(runErr, rep.Err())
}

func openJournal(ctx context.Context, cfg settings) (journal, func(), error) {
	if cfg.ledger == ledgerMemory {
		return ledger.NewMemory(), func() {}, nil
	}
	pool, err := app.OpenPool(ctx, cfg.pgDSN)
	if err != nil {
		return nil, nil, err
	}
	return ledger.NewPostgres(pool, cfg.queue), pool.Close, nil
}
