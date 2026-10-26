package main

import (
	"context"
	"log/slog"
	"os"

	"collector/services/place-de-marche/internal/config"
	"collector/services/place-de-marche/internal/migrations"
)

func migrate(ctx context.Context, logger *slog.Logger) error {
	database, err := config.LoadDatabase(os.Getenv)
	if err != nil {
		return err
	}
	return migrations.Apply(ctx, logger, database.URL)
}
