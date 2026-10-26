package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/schema"
)

const (
	binary                = "xp02-outbox"
	defaultExchange       = "pdm.evenements"
	defaultRoutingKey     = "publication.decidee.v1"
	defaultCount          = 1000
	defaultBatch          = 100
	defaultConfirmTimeout = 30 * time.Second
)

type command func(ctx context.Context, log *slog.Logger) error

var errUsage = errors.New("usage : xp02-outbox migrate|produce|relay|replay|verify")

func main() {
	log := app.Logger(binary)
	if err := run(log, os.Args[1:]); err != nil {
		log.Error("arrêt sur erreur", "erreur", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, args []string) error {
	commands := map[string]command{
		"migrate": migrate,
		"produce": produce,
		"relay":   relay,
		"replay":  replay,
		"verify":  verify,
	}
	if len(args) != 1 {
		return errUsage
	}
	selected, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("%w : commande %q inconnue", errUsage, args[0])
	}
	ctx, stop := app.SignalContext()
	defer stop()
	return selected(ctx, log.With("commande", args[0]))
}

func openPool(ctx context.Context, env *config.Env) (*pgxpool.Pool, error) {
	return app.OpenPool(ctx, env.String("XP02_PG_DSN", ""))
}

func migrate(ctx context.Context, log *slog.Logger) error {
	var env config.Env
	pool, err := openPool(ctx, &env)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := schema.Apply(ctx, pool); err != nil {
		return err
	}
	log.Info("schéma XP02 appliqué")
	return nil
}
