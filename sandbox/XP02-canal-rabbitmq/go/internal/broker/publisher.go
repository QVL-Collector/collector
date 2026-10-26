package broker

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	contentType  = "application/json"
	closeGrace   = time.Second
	returnBuffer = 64
)

type Message struct {
	Exchange      string
	RoutingKey    string
	ID            string
	Type          string
	CorrelationID string
	Timestamp     time.Time
	Body          []byte
}

type Publisher struct {
	channel        *amqp.Channel
	userID         string
	confirmTimeout time.Duration
	closed         <-chan *amqp.Error
	returned       atomic.Int64
}

func OpenPublisher(conn *amqp.Connection, userID string, confirmTimeout time.Duration) (*Publisher, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("ouverture du canal de publication : %w", err)
	}
	if err := channel.Confirm(false); err != nil {
		_ = channel.Close()
		return nil, fmt.Errorf("activation des confirmations de publication : %w", err)
	}
	publisher := &Publisher{
		channel:        channel,
		userID:         userID,
		confirmTimeout: confirmTimeout,
		closed:         channel.NotifyClose(make(chan *amqp.Error, 1)),
	}
	go publisher.countReturns(channel.NotifyReturn(make(chan amqp.Return, returnBuffer)))
	return publisher, nil
}

func (p *Publisher) Closed() <-chan *amqp.Error {
	return p.closed
}

func (p *Publisher) Returned() int64 {
	return p.returned.Load()
}

func (p *Publisher) Close() {
	_ = p.channel.Close()
}

func (p *Publisher) PublishBatch(ctx context.Context, batch []Message) ([]time.Duration, error) {
	confirmations := make([]*amqp.DeferredConfirmation, len(batch))
	sentAt := make([]time.Time, len(batch))
	for i, message := range batch {
		sentAt[i] = time.Now()
		confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(ctx, message.Exchange, message.RoutingKey, true, false, p.publishing(message))
		if err != nil {
			return nil, p.explain(fmt.Errorf("publication de %s : %w", message.ID, err))
		}
		confirmations[i] = confirmation
	}
	return p.awaitConfirmations(ctx, batch, confirmations, sentAt)
}

func (p *Publisher) awaitConfirmations(ctx context.Context, batch []Message, confirmations []*amqp.DeferredConfirmation, sentAt []time.Time) ([]time.Duration, error) {
	waitCtx, cancel := context.WithTimeout(ctx, p.confirmTimeout)
	defer cancel()
	latencies := make([]time.Duration, len(batch))
	for i, confirmation := range confirmations {
		acked, err := confirmation.WaitContext(waitCtx)
		if err != nil {
			return nil, fmt.Errorf("attente de la confirmation de %s : %w", batch[i].ID, err)
		}
		if !acked {
			return nil, p.explainNack(batch[i].ID)
		}
		latencies[i] = time.Since(sentAt[i])
	}
	return latencies, nil
}

func (p *Publisher) publishing(message Message) amqp.Publishing {
	return amqp.Publishing{
		ContentType:   contentType,
		DeliveryMode:  amqp.Persistent,
		MessageId:     message.ID,
		Type:          message.Type,
		CorrelationId: message.CorrelationID,
		Timestamp:     message.Timestamp,
		UserId:        p.userID,
		Body:          message.Body,
	}
}

func (p *Publisher) explain(cause error) error {
	if closed, reason := p.closeReason(); closed {
		return fmt.Errorf("%w : %w", cause, reason)
	}
	return cause
}

func (p *Publisher) explainNack(id string) error {
	if closed, reason := p.closeReason(); closed {
		return fmt.Errorf("confirmation de %s perdue : %w", id, reason)
	}
	return fmt.Errorf("%w : %s", ErrRejected, id)
}

func (p *Publisher) closeReason() (bool, error) {
	timer := time.NewTimer(closeGrace)
	defer timer.Stop()
	select {
	case reason := <-p.closed:
		return true, CloseReason(reason)
	case <-timer.C:
		return false, nil
	}
}

func (p *Publisher) countReturns(returns <-chan amqp.Return) {
	for range returns {
		p.returned.Add(1)
	}
}
