package stockage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/services/interactions/internal/notifications"
)

const recordQuery = `
INSERT INTO notification (message_id, vendeur_id, article_id, revision, issue, motif, decide_le)
VALUES ($1::text::uuid, $2, $3, $4, $5, $6, $7)
ON CONFLICT (message_id) DO NOTHING`

const countForSellerQuery = `SELECT count(*) FROM notification WHERE vendeur_id = $1`

const listForSellerQuery = `
SELECT id, article_id, revision, issue, motif, decide_le
FROM notification
WHERE vendeur_id = $1
ORDER BY decide_le DESC, id DESC
LIMIT $2 OFFSET $3`

var readOnlySnapshot = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Record(ctx context.Context, decision notifications.Decision) (bool, error) {
	tag, err := s.pool.Exec(ctx, recordQuery,
		decision.MessageID, uuid.UUID(decision.Seller), uuid.UUID(decision.Article), int(decision.Revision),
		string(decision.Issue), reasonColumn(decision.Reason), decision.DecidedAt,
	)
	if err != nil {
		return false, fmt.Errorf("enregistrement de la notification : %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (s *Store) ListForSeller(ctx context.Context, seller notifications.SellerID, page notifications.PageRequest) (notifications.Page, error) {
	var listed notifications.Page
	err := pgx.BeginTxFunc(ctx, s.pool, readOnlySnapshot, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, countForSellerQuery, uuid.UUID(seller)).Scan(&listed.Total); err != nil {
			return fmt.Errorf("décompte des notifications : %w", err)
		}
		rows, err := tx.Query(ctx, listForSellerQuery, uuid.UUID(seller), page.Size, page.Offset())
		if err != nil {
			return fmt.Errorf("liste des notifications : %w", err)
		}
		listed.Notifications, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (notifications.Notification, error) {
			return scanNotification(row)
		})
		return err
	})
	if err != nil {
		return notifications.Page{}, fmt.Errorf("lecture des notifications : %w", err)
	}
	return listed, nil
}

func scanNotification(row pgx.Row) (notifications.Notification, error) {
	var (
		id        uuid.UUID
		article   uuid.UUID
		revision  int
		issue     string
		reason    *string
		decidedAt time.Time
	)
	if err := row.Scan(&id, &article, &revision, &issue, &reason, &decidedAt); err != nil {
		return notifications.Notification{}, err
	}
	return notifications.Notification{
		ID:        notifications.NotificationID(id),
		Article:   notifications.ArticleID(article),
		Revision:  notifications.Revision(revision),
		Issue:     notifications.Issue(issue),
		Reason:    reasonOf(reason),
		DecidedAt: decidedAt,
	}, nil
}

func reasonColumn(reason *notifications.Reason) *string {
	if reason == nil {
		return nil
	}
	value := string(*reason)
	return &value
}

func reasonOf(raw *string) *notifications.Reason {
	if raw == nil {
		return nil
	}
	reason := notifications.Reason(*raw)
	return &reason
}
