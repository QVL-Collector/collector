package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/outbox"
	"collector/sandbox/xp02/internal/report"
)

const timeLayout = time.RFC3339

var zeroTime time.Time

func verify(ctx context.Context, log *slog.Logger) error {
	var env config.Env
	labels := app.LoadLabels(&env)
	queue := env.Required("XP02_QUEUE")
	since := env.Time("XP02_SINCE", zeroTime)
	if err := env.Err(); err != nil {
		return err
	}
	pool, err := openPool(ctx, &env)
	if err != nil {
		return err
	}
	defer pool.Close()

	coverage, err := outbox.NewStore(pool).Coverage(ctx, queue, since)
	if err != nil {
		return err
	}
	log.Info("couverture de l'outbox par les réceptions", "file", queue, "outbox", coverage.Total, "manquants", coverage.Missing)

	rep := report.New(os.Stdout, labels)
	rep.Count("outbox_total", coverage.Total, "événements", "depuis "+since.Format(timeLayout))
	rep.Count("manquants", coverage.Missing, "événements", "absents du journal de réception de "+queue)
	return rep.Err()
}
