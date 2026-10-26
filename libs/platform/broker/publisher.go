package broker

import (
	"context"
	"fmt"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	MaxBatchSize          = 256
	DefaultConfirmTimeout = 5 * time.Second
	contentType           = "application/json"
	mandatory             = true
	immediate             = false
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
	returned       <-chan amqp.Return
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
	return &Publisher{
		channel:        channel,
		userID:         userID,
		confirmTimeout: confirmTimeout,
		closed:         channel.NotifyClose(make(chan *amqp.Error, 1)),
		returned:       channel.NotifyReturn(make(chan amqp.Return, MaxBatchSize)),
	}, nil
}

func (p *Publisher) Closed() <-chan *amqp.Error {
	return p.closed
}

func (p *Publisher) Close() {
	_ = p.channel.Close()
}

func (p *Publisher) Publish(ctx context.Context, message Message) error {
	return p.PublishBatch(ctx, []Message{message})
}

func (p *Publisher) PublishBatch(ctx context.Context, batch []Message) error {
	if len(batch) > MaxBatchSize {
		return fmt.Errorf("lot de %d messages pour %d autorisés", len(batch), MaxBatchSize)
	}
	p.drainReturned()
	confirmations := make([]*amqp.DeferredConfirmation, len(batch))
	for i, message := range batch {
		confirmation, err := p.channel.PublishWithDeferredConfirmWithContext(ctx, message.Exchange, message.RoutingKey, mandatory, immediate, p.publishing(message))
		if err != nil {
			return fmt.Errorf("publication de %s : %w", message.ID, err)
		}
		confirmations[i] = confirmation
	}
	if err := p.awaitConfirmations(ctx, batch, confirmations); err != nil {
		return err
	}
	if unroutable := p.drainReturned(); len(unroutable) > 0 {
		return fmt.Errorf("%w : %s", ErrUnroutable, strings.Join(unroutable, ", "))
	}
	return nil
}

func (p *Publisher) awaitConfirmations(ctx context.Context, batch []Message, confirmations []*amqp.DeferredConfirmation) error {
	waitCtx, cancel := context.WithTimeout(ctx, p.confirmTimeout)
	defer cancel()
	for i, confirmation := range confirmations {
		acked, err := confirmation.WaitContext(waitCtx)
		if err != nil {
			return fmt.Errorf("attente de la confirmation de %s : %w", batch[i].ID, err)
		}
		if !acked {
			return fmt.Errorf("%w : %s", ErrRejected, batch[i].ID)
		}
	}
	return nil
}

func (p *Publisher) drainReturned() []string {
	var ids []string
	for {
		select {
		case returned, open := <-p.returned:
			if !open {
				return ids
			}
			ids = append(ids, returned.MessageId)
		default:
			return ids
		}
	}
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
