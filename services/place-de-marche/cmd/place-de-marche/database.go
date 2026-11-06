package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/libs/platform/database"
)

func withDatabase(ctx context.Context, use func(pool *pgxpool.Pool) error) error {
	databaseConfig, err := database.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
	pool, err := database.Connect(ctx, databaseConfig.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return use(pool)
}
