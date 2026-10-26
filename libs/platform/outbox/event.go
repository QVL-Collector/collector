package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"collector/libs/platform/broker"
)

const insertEventQuery = `
INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, corps)
VALUES ($1::text::uuid, $2, $3, $4, $5::text::uuid, $6)`

type Event struct {
	ID            string
	Exchange      string
	RoutingKey    string
	Type          string
	CorrelationID string
	Body          []byte
	CreatedAt     time.Time
}

func Enqueue(ctx context.Context, tx pgx.Tx, event Event) error {
	_, err := tx.Exec(ctx, insertEventQuery, event.ID, event.Exchange, event.RoutingKey, event.Type, event.CorrelationID, event.Body)
	if err != nil {
		return fmt.Errorf("écriture de %s dans l'outbox : %w", event.ID, err)
	}
	return nil
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
