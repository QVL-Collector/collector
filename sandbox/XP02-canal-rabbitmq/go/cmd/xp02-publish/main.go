package main

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/ident"
	"collector/sandbox/xp02/internal/report"
)

const (
	binary         = "xp02-publish"
	noPoison       = -1
	defaultType    = "controle.demande"
	defaultCount   = 1000
	defaultBatch   = 50
	defaultTimeout = 30 * time.Second
)

type settings struct {
	amqp           app.AMQP
	labels         report.Labels
	exchange       string
	routingKey     string
	eventType      string
	userID         string
	count          int
	batchSize      int
	rate           int
	poisonAt       int
	confirmTimeout time.Duration
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
		amqp:           app.LoadAMQP(&env, binary),
		labels:         app.LoadLabels(&env),
		exchange:       env.Required("XP02_EXCHANGE"),
		routingKey:     env.Required("XP02_ROUTING_KEY"),
		eventType:      env.String("XP02_EVENT_TYPE", defaultType),
		count:          env.Int("XP02_COUNT", defaultCount),
		batchSize:      env.Int("XP02_BATCH", defaultBatch),
		rate:           env.Int("XP02_RATE", 0),
		poisonAt:       env.Int("XP02_POISON_AT", noPoison),
		confirmTimeout: env.Duration("XP02_CONFIRM_TIMEOUT", defaultTimeout),
	}
	cfg.userID = env.String("XP02_USER_ID", cfg.amqp.User)
	env.Require(cfg.count > 0, "XP02_COUNT doit être strictement positif")
	env.Require(cfg.batchSize > 0, "XP02_BATCH doit être strictement positif")
	env.Require(cfg.rate >= 0, "XP02_RATE doit être positif ou nul")
	env.Require(cfg.poisonAt >= noPoison && cfg.poisonAt <= cfg.count, "XP02_POISON_AT doit être compris entre -1 et XP02_COUNT")
	env.Require(cfg.confirmTimeout > 0, "XP02_CONFIRM_TIMEOUT doit être strictement positif")
	return cfg, env.Err()
}

func run(log *slog.Logger) error {
	cfg, err := load()
	if err != nil {
		return err
	}
	correlationID, err := ident.New()
	if err != nil {
		return err
	}
	ctx, stop := app.SignalContext()
	defer stop()

	rep := report.New(os.Stdout, cfg.labels)
	w := newWave(cfg, correlationID, log)
	runErr := app.Supervisor(cfg.amqp, log, rep).Run(ctx, w.worker)

	w.report(rep)
	app.ReportFault(rep, runErr)
	return errors.Join(runErr, rep.Err())
}
