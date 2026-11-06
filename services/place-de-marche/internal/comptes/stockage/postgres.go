package stockage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/services/place-de-marche/internal/comptes"
)

const (
	enrollQuery = `INSERT INTO compte (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`
	statusQuery = `SELECT inscrit_le, suspendu FROM compte WHERE id = $1`
)

const setSuspensionQuery = `
UPDATE compte
SET suspendu = $2, suspendu_le = CASE WHEN $2 THEN coalesce(suspendu_le, now()) END
WHERE id = $1`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Enroll(ctx context.Context, tx pgx.Tx, id comptes.AccountID) error {
	if _, err := tx.Exec(ctx, enrollQuery, uuid.UUID(id)); err != nil {
		return fmt.Errorf("inscription du compte : %w", err)
	}
	return nil
}

func (s *Store) Status(ctx context.Context, tx pgx.Tx, id comptes.AccountID) (comptes.Status, error) {
	var (
		registeredAt time.Time
		suspended    bool
	)
	err := tx.QueryRow(ctx, statusQuery, uuid.UUID(id)).Scan(&registeredAt, &suspended)
	if errors.Is(err, pgx.ErrNoRows) {
		return comptes.Status{}, comptes.ErrAccountNotFound
	}
	if err != nil {
		return comptes.Status{}, fmt.Errorf("lecture du statut du compte : %w", err)
	}
	return comptes.Status{RegisteredAt: registeredAt, Suspended: suspended}, nil
}

func (s *Store) SetSuspension(ctx context.Context, id comptes.AccountID, suspended bool) error {
	tag, err := s.pool.Exec(ctx, setSuspensionQuery, uuid.UUID(id), suspended)
	if err != nil {
		return fmt.Errorf("suspension du compte : %w", err)
	}
	if tag.RowsAffected() == 0 {
		return comptes.ErrAccountNotFound
	}
	return nil
}
