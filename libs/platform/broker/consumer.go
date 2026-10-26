package broker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const DefaultPrefetch = 10

type Disposition int

const (
	Ack Disposition = iota
	Reject
)

type Delivery struct {
	MessageID   string
	UserID      string
	Redelivered bool
	Body        []byte
}

type Handler func(ctx context.Context, delivery Delivery) (Disposition, error)

type Consumer struct {
	Queue             string
	ExpectedPublisher string
	Prefetch          int
	Backoff           Backoff
	Logger            *slog.Logger
}

func (c Consumer) Worker(handle Handler) Worker {
	return func(ctx context.Context, conn *amqp.Connection) error {
		return c.consume(ctx, conn, handle)
	}
}

func (c Consumer) WorkerWithPublisher(userID string, handlerFor func(*Publisher) Handler) Worker {
	return func(ctx context.Context, conn *amqp.Connection) error {
		publisher, err := OpenPublisher(conn, userID, DefaultConfirmTimeout)
		if err != nil {
			return err
		}
		defer publisher.Close()
		return c.consume(ctx, conn, handlerFor(publisher))
	}
}

func (c Consumer) consume(ctx context.Context, conn *amqp.Connection, handle Handler) error {
	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("ouverture du canal de consommation : %w", err)
	}
	defer func() { _ = channel.Close() }()
	closed := channel.NotifyClose(make(chan *amqp.Error, 1))
	if err := channel.Qos(c.Prefetch, 0, false); err != nil {
		return fmt.Errorf("réglage du préchargement : %w", err)
	}
	deliveries, err := channel.Consume(c.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consommation de %s : %w", c.Queue, err)
	}
	return c.loop(ctx, handle, deliveries, closed)
}

func (c Consumer) loop(ctx context.Context, handle Handler, deliveries <-chan amqp.Delivery, closed <-chan *amqp.Error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case reason := <-closed:
			return CloseReason(reason)
		case delivery, open := <-deliveries:
			if !open {
				return fmt.Errorf("%w : consommation de %s interrompue", ErrConnectionLost, c.Queue)
			}
			if err := c.process(ctx, handle, delivery, closed); err != nil {
				return err
			}
		}
	}
}

func (c Consumer) process(ctx context.Context, handle Handler, delivery amqp.Delivery, closed <-chan *amqp.Error) error {
	if delivery.UserId != c.ExpectedPublisher {
		c.Logger.WarnContext(ctx, "émetteur inattendu, message écarté",
			slog.String("event", "canal.emetteur_refuse"),
			slog.String("file", c.Queue),
			slog.String("message_id", delivery.MessageId),
			slog.String("emetteur", delivery.UserId),
		)
		return settle(delivery, Reject)
	}
	received := Delivery{MessageID: delivery.MessageId, UserID: delivery.UserId, Redelivered: delivery.Redelivered, Body: delivery.Body}
	for attempt := 0; ; attempt++ {
		disposition, err := handle(ctx, received)
		if err == nil {
			return settle(delivery, disposition)
		}
		if IsFault(err) {
			return err
		}
		c.Logger.WarnContext(ctx, "échec transitoire, message conservé sans acquittement",
			slog.String("event", "canal.echec_transitoire"),
			slog.String("file", c.Queue),
			slog.String("message_id", delivery.MessageId),
			slog.Any("erreur", err),
		)
		if err := wait(ctx, closed, c.Backoff.Delay(attempt)); err != nil {
			return err
		}
	}
}

func wait(ctx context.Context, closed <-chan *amqp.Error, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case reason := <-closed:
		return CloseReason(reason)
	case <-timer.C:
		return nil
	}
}

func settle(delivery amqp.Delivery, disposition Disposition) error {
	var err error
	switch disposition {
	case Ack:
		err = delivery.Ack(false)
	case Reject:
		err = delivery.Nack(false, false)
	}
	if err != nil {
		return fmt.Errorf("acquittement de %s : %w", delivery.MessageId, err)
	}
	return nil
}
