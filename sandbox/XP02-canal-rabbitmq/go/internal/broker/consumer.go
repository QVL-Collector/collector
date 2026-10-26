package broker

import (
	"context"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Disposition int

const (
	Ack Disposition = iota
	Reject
	Requeue
)

type Handler func(ctx context.Context, delivery amqp.Delivery) (Disposition, error)

type Consumer struct {
	Queue    string
	Prefetch int
	Backoff  Backoff
	Log      *slog.Logger
}

func (c Consumer) Run(handle Handler) Worker {
	return func(ctx context.Context, conn *amqp.Connection) error {
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
				return CloseReason(<-closed)
			}
			if err := c.process(ctx, handle, delivery); err != nil {
				return err
			}
		}
	}
}

func (c Consumer) process(ctx context.Context, handle Handler, delivery amqp.Delivery) error {
	for attempt := 0; ; attempt++ {
		disposition, err := handle(ctx, delivery)
		if err == nil {
			return settle(delivery, disposition)
		}
		c.Log.Warn("échec transitoire, message conservé sans acquittement", "message_id", delivery.MessageId, "erreur", err)
		if !Pause(ctx, c.Backoff.Delay(attempt)) {
			return ctx.Err()
		}
	}
}

func settle(delivery amqp.Delivery, disposition Disposition) error {
	var err error
	switch disposition {
	case Reject:
		err = delivery.Nack(false, false)
	case Requeue:
		err = delivery.Nack(false, true)
	case Ack:
		err = delivery.Ack(false)
	}
	if err != nil {
		return fmt.Errorf("acquittement de %s : %w", delivery.MessageId, err)
	}
	return nil
}
