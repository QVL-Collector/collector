package schema

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ddl = `
CREATE TABLE IF NOT EXISTS xp02_decision_publication (
    decision_id uuid PRIMARY KEY,
    article_id uuid NOT NULL,
    revision integer NOT NULL CHECK (revision > 0),
    issue text NOT NULL CHECK (issue IN ('publie', 'refuse', 'a_verifier')),
    decided_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS xp02_outbox (
    id uuid PRIMARY KEY,
    exchange text NOT NULL,
    routing_key text NOT NULL,
    message_type text NOT NULL,
    correlation_id uuid NOT NULL,
    body bytea NOT NULL,
    created_at timestamptz NOT NULL,
    published_at timestamptz
);

CREATE INDEX IF NOT EXISTS xp02_outbox_pending
    ON xp02_outbox (created_at, id) WHERE published_at IS NULL;

CREATE INDEX IF NOT EXISTS xp02_outbox_published
    ON xp02_outbox (published_at) WHERE published_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS xp02_reception (
    queue text NOT NULL,
    event_id uuid NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (queue, event_id)
);
`

func Apply(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("application du schéma XP02 : %w", err)
	}
	return nil
}
