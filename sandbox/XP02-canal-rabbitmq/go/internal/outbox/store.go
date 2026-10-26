package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/sandbox/xp02/internal/broker"
	"collector/sandbox/xp02/internal/envelope"
)

const (
	insertEvent = `
INSERT INTO xp02_outbox (id, exchange, routing_key, message_type, correlation_id, body, created_at)
VALUES ($1::text::uuid, $2, $3, $4, $5::text::uuid, $6, $7)`

	selectPendingForUpdate = `
SELECT id::text, exchange, routing_key, message_type, correlation_id::text, body, created_at
FROM xp02_outbox
WHERE published_at IS NULL
ORDER BY created_at, id
LIMIT $1
FOR UPDATE SKIP LOCKED`

	selectPublishedWithin = `
SELECT id::text, exchange, routing_key, message_type, correlation_id::text, body, created_at
FROM xp02_outbox
WHERE published_at >= $1 AND published_at < $2
ORDER BY created_at, id
LIMIT $3`

	updatePublished = `UPDATE xp02_outbox SET published_at = now() WHERE id = ANY($1::text[]::uuid[])`

	selectBacklog = `
SELECT count(*), coalesce(extract(epoch FROM now() - min(created_at)), 0)::float8
FROM xp02_outbox
WHERE published_at IS NULL`

	selectMissing = `
SELECT count(*), count(*) FILTER (WHERE r.event_id IS NULL)
FROM xp02_outbox o
LEFT JOIN xp02_reception r ON r.event_id = o.id AND r.queue = $1
WHERE o.created_at >= $2`

	selectServerNow = `SELECT now()`
)

type Event struct {
	ID            string
	Exchange      string
	RoutingKey    string
	Type          string
	CorrelationID string
	Body          []byte
	CreatedAt     time.Time
}

func FromEnvelope(env envelope.Envelope, exchange, routingKey string) (Event, error) {
	body, err := env.Encode()
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:            env.ID,
		Exchange:      exchange,
		RoutingKey:    routingKey,
		Type:          env.Type,
		CorrelationID: env.CorrelationID,
		Body:          body,
		CreatedAt:     env.EmittedAt,
	}, nil
}

func (e Event) message() broker.Message {
	return broker.Message{
		Exchange:      e.Exchange,
		RoutingKey:    e.RoutingKey,
		ID:            e.ID,
		Type:          e.Type,
		CorrelationID: e.CorrelationID,
		Timestamp:     e.CreatedAt,
		Body:          e.Body,
	}
}

type Backlog struct {
	Count  int64
	Oldest time.Duration
}

type Coverage struct {
	Total   int64
	Missing int64
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func Enqueue(ctx context.Context, tx pgx.Tx, event Event) error {
	_, err := tx.Exec(ctx, insertEvent, event.ID, event.Exchange, event.RoutingKey, event.Type, event.CorrelationID, event.Body, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("écriture de %s dans l'outbox : %w", event.ID, err)
	}
	return nil
}

func (s *Store) relayBatch(ctx context.Context, limit int, publish func([]Event) error) (int, error) {
	relayed := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		events, err := queryEvents(ctx, tx, selectPendingForUpdate, limit)
		if err != nil || len(events) == 0 {
			return err
		}
		if err := publish(events); err != nil {
			return err
		}
		relayed = len(events)
		return markPublished(ctx, tx, events)
	})
	if err != nil {
		return 0, fmt.Errorf("relais d'un lot de l'outbox : %w", err)
	}
	return relayed, nil
}

func (s *Store) publishedWithin(ctx context.Context, since, until time.Time, limit int) ([]Event, error) {
	return queryEvents(ctx, s.pool, selectPublishedWithin, since, until, limit)
}

func (s *Store) Backlog(ctx context.Context) (Backlog, error) {
	var count int64
	var seconds float64
	if err := s.pool.QueryRow(ctx, selectBacklog).Scan(&count, &seconds); err != nil {
		return Backlog{}, fmt.Errorf("lecture de l'arriéré de l'outbox : %w", err)
	}
	return Backlog{Count: count, Oldest: time.Duration(seconds * float64(time.Second))}, nil
}

func (s *Store) Coverage(ctx context.Context, queue string, since time.Time) (Coverage, error) {
	var coverage Coverage
	if err := s.pool.QueryRow(ctx, selectMissing, queue, since).Scan(&coverage.Total, &coverage.Missing); err != nil {
		return Coverage{}, fmt.Errorf("comparaison de l'outbox et des réceptions de %s : %w", queue, err)
	}
	return coverage, nil
}

func (s *Store) ServerNow(ctx context.Context) (time.Time, error) {
	var now time.Time
	if err := s.pool.QueryRow(ctx, selectServerNow).Scan(&now); err != nil {
		return time.Time{}, fmt.Errorf("lecture de l'heure du serveur PostgreSQL : %w", err)
	}
	return now, nil
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func queryEvents(ctx context.Context, db querier, sql string, args ...any) ([]Event, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("lecture de l'outbox : %w", err)
	}
	events, err := pgx.CollectRows(rows, scanEvent)
	if err != nil {
		return nil, fmt.Errorf("lecture de l'outbox : %w", err)
	}
	return events, nil
}

func scanEvent(row pgx.CollectableRow) (Event, error) {
	var event Event
	err := row.Scan(&event.ID, &event.Exchange, &event.RoutingKey, &event.Type, &event.CorrelationID, &event.Body, &event.CreatedAt)
	return event, err
}

func markPublished(ctx context.Context, db execer, events []Event) error {
	if _, err := db.Exec(ctx, updatePublished, ids(events)); err != nil {
		return fmt.Errorf("marquage des événements publiés : %w", err)
	}
	return nil
}

func ids(events []Event) []string {
	result := make([]string, len(events))
	for i, event := range events {
		result[i] = event.ID
	}
	return result
}

func messages(events []Event) []broker.Message {
	result := make([]broker.Message, len(events))
	for i, event := range events {
		result[i] = event.message()
	}
	return result
}
