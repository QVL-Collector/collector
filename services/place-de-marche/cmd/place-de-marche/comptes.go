package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/libs/platform/logging"
	"collector/services/place-de-marche/internal/comptes"
	comptesstockage "collector/services/place-de-marche/internal/comptes/stockage"
)

const operatorAuthor = "exploitation"

var errMissingAccountID = errors.New("identifiant de compte attendu, et lui seul")

func setSuspension(ctx context.Context, logger *slog.Logger, operands []string, suspended bool) error {
	id, err := accountIDOf(operands)
	if err != nil {
		return err
	}
	ctx = logging.StartOperation(ctx)
	return withDatabase(ctx, func(pool *pgxpool.Pool) error {
		if err := comptesstockage.NewStore(pool).SetSuspension(ctx, id, suspended); err != nil {
			return fmt.Errorf("compte %s : %w", id, err)
		}
		logging.Audit(ctx, logger, "suspension du compte modifiée", "comptes.suspension",
			slog.String("compte_id", id.String()),
			slog.String("auteur", operatorAuthor),
			slog.String("action", suspensionAction(suspended)),
			slog.Bool("suspendu", suspended),
		)
		return nil
	})
}

func suspensionAction(suspended bool) string {
	if suspended {
		return suspendCommand
	}
	return reinstateCommand
}

func accountIDOf(operands []string) (comptes.AccountID, error) {
	if len(operands) != 1 {
		return comptes.AccountID{}, errMissingAccountID
	}
	return comptes.ParseAccountID(operands[0])
}
