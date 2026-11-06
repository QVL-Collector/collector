package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func ApplyMigrations(ctx context.Context, logger *slog.Logger, databaseURL string, scripts fs.FS) (err error) {
	connConfig, parseErr := pgx.ParseConfig(databaseURL)
	if parseErr != nil {
		return errors.New("configuration de la base de données illisible")
	}
	db := stdlib.OpenDB(*connConfig)
	defer func() {
		err = errors.Join(err, db.Close())
	}()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, scripts)
	if err != nil {
		return fmt.Errorf("préparation des migrations : %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("application des migrations : %w", err)
	}
	logger.InfoContext(ctx, "migrations appliquées",
		slog.String("event", "base.migrations"),
		slog.Int("nombre", len(results)),
	)
	return nil
}
