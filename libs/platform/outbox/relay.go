package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"collector/libs/platform/broker"
)

const (
	pollInterval  = 500 * time.Millisecond
	batchSize     = 50
	blockedBuffer = 8
)

const selectPendingQuery = `
SELECT id::text, echange, cle_routage, type_message, correlation_id::text, corps, cree_le
FROM outbox
WHERE publie_le IS NULL
ORDER BY cree_le, id
LIMIT $1
FOR UPDATE SKIP LOCKED`

const markPublishedQuery = `UPDATE outbox SET publie_le = now() WHERE id = ANY($1::text[]::uuid[])`

type Relay struct {
	pool   *pgxpool.Pool
	userID string
	logger *slog.Logger
}

func NewRelay(pool *pgxpool.Pool, userID string, logger *slog.Logger) *Relay {
	return &Relay{pool: pool, userID: userID, logger: logger}
}

func (r *Relay) Worker() broker.Worker {
	return func(ctx context.Context, conn *amqp.Connection) error {
		publisher, err := broker.OpenPublisher(conn, r.userID, broker.DefaultConfirmTimeout)
		if err != nil {
			return err
		}
		defer publisher.Close()
		return r.loop(ctx, conn, publisher)
	}
}

func (r *Relay) loop(ctx context.Context, conn *amqp.Connection, publisher *broker.Publisher) error {
	connectionClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
	blocked := conn.NotifyBlocked(make(chan amqp.Blocking, blockedBuffer))
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	paused := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case reason := <-connectionClosed:
			return broker.CloseReason(reason)
		case reason := <-publisher.Closed():
			return broker.CloseReason(reason)
		case state, open := <-blocked:
			if !open {
				return broker.ErrConnectionLost
			}
			paused = state.Active
			r.logger.WarnContext(ctx, "blocage des producteurs signalé par le broker",
				slog.String("event", "outbox.blocage"),
				slog.Bool("actif", state.Active),
				slog.String("raison", state.Reason),
			)
		case <-ticker.C:
			if paused {
				continue
			}
			if err := r.drain(ctx, publisher); err != nil {
				return err
			}
		}
	}
}

func (r *Relay) drain(ctx context.Context, publisher *broker.Publisher) error {
	for {
		relayed, err := r.relayBatch(ctx, publisher)
		if broker.IsFault(err) {
			return err
		}
		if err != nil {
			r.logger.WarnContext(ctx, "lot de l'outbox non publié, nouvel essai au prochain cycle",
				slog.String("event", "outbox.lot_non_publie"),
				slog.Any("erreur", err),
			)
			return nil
		}
		if relayed < batchSize {
			return nil
		}
	}
}

func (r *Relay) relayBatch(ctx context.Context, publisher *broker.Publisher) (int, error) {
	relayed := 0
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		events, err := pendingEvents(ctx, tx)
		if err != nil || len(events) == 0 {
			return err
		}
		if err := publisher.PublishBatch(ctx, messages(events)); err != nil {
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

func pendingEvents(ctx context.Context, tx pgx.Tx) ([]Event, error) {
	rows, err := tx.Query(ctx, selectPendingQuery, batchSize)
	if err != nil {
		return nil, fmt.Errorf("lecture de l'outbox : %w", err)
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Event, error) {
		var event Event
		err := row.Scan(&event.ID, &event.Exchange, &event.RoutingKey, &event.Type, &event.CorrelationID, &event.Body, &event.CreatedAt)
		return event, err
	})
	if err != nil {
		return nil, fmt.Errorf("lecture de l'outbox : %w", err)
	}
	return events, nil
}

func markPublished(ctx context.Context, tx pgx.Tx, events []Event) error {
	ids := make([]string, len(events))
	for i, event := range events {
		ids[i] = event.ID
	}
	if _, err := tx.Exec(ctx, markPublishedQuery, ids); err != nil {
		return fmt.Errorf("marquage des événements publiés : %w", err)
	}
	return nil
}

func messages(events []Event) []broker.Message {
	result := make([]broker.Message, len(events))
	for i, event := range events {
		result[i] = event.message()
	}
	return result
}
