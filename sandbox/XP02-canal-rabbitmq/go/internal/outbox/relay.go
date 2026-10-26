package outbox

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"collector/sandbox/xp02/internal/broker"
	"collector/sandbox/xp02/internal/stats"
)

const blockedBuffer = 8

type RelayConfig struct {
	Store          *Store
	BatchSize      int
	Interval       time.Duration
	UserID         string
	ConfirmTimeout time.Duration
	Log            *slog.Logger
	OnPublished    func(count int)
}

type Relay struct {
	cfg       RelayConfig
	lags      stats.Latencies
	published atomic.Int64
}

func NewRelay(cfg RelayConfig) *Relay {
	return &Relay{cfg: cfg}
}

func (r *Relay) Published() int64 {
	return r.published.Load()
}

func (r *Relay) Lags() stats.Summary {
	return r.lags.Summary()
}

func (r *Relay) Worker() broker.Worker {
	return func(ctx context.Context, conn *amqp.Connection) error {
		publisher, err := broker.OpenPublisher(conn, r.cfg.UserID, r.cfg.ConfirmTimeout)
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
	ticker := time.NewTicker(r.cfg.Interval)
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
			r.cfg.Log.Warn("blocage des producteurs signalé par le broker", "actif", state.Active, "raison", state.Reason)
		case <-ticker.C:
			if paused {
				continue
			}
			if err := r.drain(ctx, publisher); err != nil {
				if !errors.Is(err, broker.ErrRejected) {
					return err
				}
				r.cfg.Log.Warn("lot refusé par le broker, nouvel essai au prochain cycle", "erreur", err)
			}
		}
	}
}

func (r *Relay) drain(ctx context.Context, publisher *broker.Publisher) error {
	for {
		relayed, err := r.cfg.Store.relayBatch(ctx, r.cfg.BatchSize, func(events []Event) error {
			return r.publish(ctx, publisher, events)
		})
		if err != nil {
			return err
		}
		if relayed < r.cfg.BatchSize {
			return nil
		}
	}
}

func (r *Relay) publish(ctx context.Context, publisher *broker.Publisher, events []Event) error {
	if _, err := publisher.PublishBatch(ctx, messages(events)); err != nil {
		return err
	}
	confirmedAt := time.Now()
	for _, event := range events {
		r.lags.Add(confirmedAt.Sub(event.CreatedAt))
	}
	r.published.Add(int64(len(events)))
	if r.cfg.OnPublished != nil {
		r.cfg.OnPublished(len(events))
	}
	return nil
}
