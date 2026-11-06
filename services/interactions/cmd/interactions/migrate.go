package main

import (
	"context"
	"log/slog"
	"os"

	"collector/libs/platform/database"
	"collector/services/interactions/internal/migrations"
)

func migrate(ctx context.Context, logger *slog.Logger) error {
	databaseConfig, err := database.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
	return migrations.Apply(ctx, logger, databaseConfig.URL)
}
