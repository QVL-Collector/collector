package migrations

import (
	"context"
	"embed"
	"log/slog"

	"collector/libs/platform/database"
)

//go:embed *.sql
var scripts embed.FS

func Apply(ctx context.Context, logger *slog.Logger, databaseURL string) error {
	return database.ApplyMigrations(ctx, logger, databaseURL, scripts)
}
