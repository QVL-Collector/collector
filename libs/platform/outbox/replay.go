package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	withinRetentionQuery = `SELECT $1::timestamptz >= now() - conservation_outbox(), conservation_outbox()::text`
	replayQuery          = `UPDATE outbox SET publie_le = NULL WHERE publie_le >= $1 AND publie_le < $2`
)

var errInvalidWindow = errors.New("fenêtre de rejeu invalide")

type Window struct {
	From time.Time
	To   time.Time
}

func NewWindow(from, to, now time.Time) (Window, error) {
	if !from.Before(to) {
		return Window{}, fmt.Errorf("%w : le début doit précéder la fin", errInvalidWindow)
	}
	if to.After(now) {
		return Window{}, fmt.Errorf("%w : la fin est dans le futur", errInvalidWindow)
	}
	return Window{From: from, To: to}, nil
}

func Replay(ctx context.Context, pool *pgxpool.Pool, window Window) (int64, error) {
	var replayed int64
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := checkRetention(ctx, tx, window); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, replayQuery, window.From, window.To)
		replayed = tag.RowsAffected()
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("rejeu de l'outbox : %w", err)
	}
	return replayed, nil
}

func checkRetention(ctx context.Context, tx pgx.Tx, window Window) error {
	var (
		retained  bool
		retention string
	)
	if err := tx.QueryRow(ctx, withinRetentionQuery, window.From).Scan(&retained, &retention); err != nil {
		return fmt.Errorf("lecture de la conservation de l'outbox : %w", err)
	}
	if !retained {
		return fmt.Errorf("%w : le début dépasse la conservation de %s", errInvalidWindow, retention)
	}
	return nil
}
