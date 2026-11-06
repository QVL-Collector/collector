package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const replayQuery = `UPDATE outbox SET publie_le = NULL WHERE publie_le >= $1 AND publie_le < $2`

var ErrInvalidWindow = errors.New("fenêtre de rejeu invalide")

type Window struct {
	From time.Time
	To   time.Time
}

func NewWindow(from, to, now time.Time) (Window, error) {
	if !from.Before(to) {
		return Window{}, fmt.Errorf("%w : le début doit précéder la fin", ErrInvalidWindow)
	}
	if to.After(now) {
		return Window{}, fmt.Errorf("%w : la fin est dans le futur", ErrInvalidWindow)
	}
	if from.Before(now.Add(-Retention)) {
		return Window{}, fmt.Errorf("%w : le début dépasse la conservation de %s", ErrInvalidWindow, Retention)
	}
	return Window{From: from, To: to}, nil
}

func Replay(ctx context.Context, pool *pgxpool.Pool, window Window) (int64, error) {
	tag, err := pool.Exec(ctx, replayQuery, window.From, window.To)
	if err != nil {
		return 0, fmt.Errorf("rejeu de l'outbox : %w", err)
	}
	return tag.RowsAffected(), nil
}
