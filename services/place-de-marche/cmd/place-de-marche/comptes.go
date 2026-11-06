package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/services/place-de-marche/internal/comptes"
	comptesstockage "collector/services/place-de-marche/internal/comptes/stockage"
)

var errMissingAccountID = errors.New("identifiant de compte attendu, et lui seul")

func setSuspension(ctx context.Context, logger *slog.Logger, operands []string, suspended bool) error {
	id, err := accountIDOf(operands)
	if err != nil {
		return err
	}
	return withDatabase(ctx, func(pool *pgxpool.Pool) error {
		if err := comptesstockage.NewStore(pool).SetSuspension(ctx, id, suspended); err != nil {
			return fmt.Errorf("compte %s : %w", id, err)
		}
		logger.InfoContext(ctx, "suspension du compte modifiée",
			slog.String("event", "comptes.suspension"),
			slog.String("compte_id", id.String()),
			slog.Bool("suspendu", suspended),
		)
		return nil
	})
}

func accountIDOf(operands []string) (comptes.AccountID, error) {
	if len(operands) != 1 {
		return comptes.AccountID{}, errMissingAccountID
	}
	return comptes.ParseAccountID(operands[0])
}
