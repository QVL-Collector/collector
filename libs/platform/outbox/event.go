package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"collector/libs/platform/broker"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
)

const insertEventQuery = `
INSERT INTO outbox (id, echange, cle_routage, type_message, correlation_id, traceparent, corps)
VALUES ($1::text::uuid, $2, $3, $4, $5::text::uuid, NULLIF($6::text, ''), $7)`

type Event struct {
	ID            string
	Exchange      string
	RoutingKey    string
	Type          string
	CorrelationID string
	TraceParent   string
	Body          []byte
	CreatedAt     time.Time
}

func NewEvent(ctx context.Context, envelope message.Envelope, exchange, routingKey string, body []byte) Event {
	traceParent, _ := logging.TraceParent(ctx)
	return Event{
		ID:            envelope.ID,
		Exchange:      exchange,
		RoutingKey:    routingKey,
		Type:          envelope.Type,
		CorrelationID: envelope.CorrelationID,
		TraceParent:   traceParent,
		Body:          body,
		CreatedAt:     envelope.EmittedAt,
	}
}

func Enqueue(ctx context.Context, tx pgx.Tx, event Event) error {
	_, err := tx.Exec(ctx, insertEventQuery, event.ID, event.Exchange, event.RoutingKey, event.Type, event.CorrelationID, event.TraceParent, event.Body)
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
		TraceParent:   e.TraceParent,
		Timestamp:     e.CreatedAt,
		Body:          e.Body,
	}
}
