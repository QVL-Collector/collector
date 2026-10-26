package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/ident"
	"collector/sandbox/xp02/internal/outbox"
	"collector/sandbox/xp02/internal/report"
)

func replay(ctx context.Context, log *slog.Logger) error {
	var env config.Env
	amqpSettings := app.LoadAMQP(&env, binary+"-replay")
	labels := app.LoadLabels(&env)
	since := env.Time("XP02_REPLAY_SINCE", zeroTime)
	batchSize := env.Int("XP02_BATCH", defaultBatch)
	confirmTimeout := env.Duration("XP02_CONFIRM_TIMEOUT", defaultConfirmTimeout)
	env.Require(!since.IsZero(), "XP02_REPLAY_SINCE est obligatoire (RFC 3339)")
	env.Require(batchSize > 0, "XP02_BATCH doit être strictement positif")
	if err := env.Err(); err != nil {
		return err
	}
	pool, err := openPool(ctx, &env)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := outbox.NewStore(pool)
	until, err := store.ServerNow(ctx)
	if err != nil {
		return err
	}
	replayer, err := outbox.NewReplay(outbox.ReplayConfig{
		Store:          store,
		BatchSize:      batchSize,
		Since:          since,
		Until:          until,
		UserID:         amqpSettings.User,
		ConfirmTimeout: confirmTimeout,
	})
	if err != nil {
		return err
	}
	auditID, err := ident.New()
	if err != nil {
		return err
	}
	log.Info("rejeu de l'outbox demandé", "audit_id", auditID, "depuis", since, "jusqu_a", until)

	rep := report.New(os.Stdout, labels)
	runErr := app.Supervisor(amqpSettings, log, rep).Run(ctx, replayer.Worker())

	log.Info("rejeu de l'outbox terminé", "audit_id", auditID, "republies", replayer.Replayed(), "erreur", runErr)
	window := fmt.Sprintf("fenêtre %s → %s, audit %s", since.Format(timeLayout), until.Format(timeLayout), auditID)
	rep.Count("republies", replayer.Replayed(), "messages", window)
	app.ReportFault(rep, runErr)
	return errors.Join(runErr, rep.Err())
}
