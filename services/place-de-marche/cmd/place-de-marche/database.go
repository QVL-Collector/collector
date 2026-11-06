package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/services/place-de-marche/internal/annonces/stockage"
	"collector/services/place-de-marche/internal/config"
)

func withDatabase(ctx context.Context, use func(pool *pgxpool.Pool) error) error {
	database, err := config.LoadDatabase(os.Getenv)
	if err != nil {
		return err
	}
	pool, err := stockage.Connect(ctx, database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return use(pool)
}
