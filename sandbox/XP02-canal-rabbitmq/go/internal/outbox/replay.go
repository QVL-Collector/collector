package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"collector/sandbox/xp02/internal/broker"
)

const MaxReplayWindow = 7 * 24 * time.Hour

var ErrReplayWindow = errors.New("fenêtre de rejeu invalide")

type ReplayConfig struct {
	Store          *Store
	BatchSize      int
	Since          time.Time
	Until          time.Time
	UserID         string
	ConfirmTimeout time.Duration
}

type Replay struct {
	cfg      ReplayConfig
	replayed int64
}

func NewReplay(cfg ReplayConfig) (*Replay, error) {
	window := cfg.Until.Sub(cfg.Since)
	switch {
	case window <= 0:
		return nil, fmt.Errorf("%w : %s n'est pas antérieur à %s", ErrReplayWindow, cfg.Since, cfg.Until)
	case window > MaxReplayWindow:
		return nil, fmt.Errorf("%w : %s dépasse le plafond de %s", ErrReplayWindow, window, MaxReplayWindow)
	}
	return &Replay{cfg: cfg}, nil
}

func (r *Replay) Replayed() int64 {
	return r.replayed
}

func (r *Replay) Worker() broker.Worker {
	return func(ctx context.Context, conn *amqp.Connection) error {
		publisher, err := broker.OpenPublisher(conn, r.cfg.UserID, r.cfg.ConfirmTimeout)
		if err != nil {
			return err
		}
		defer publisher.Close()
		for {
			done, err := r.replayBatch(ctx, publisher)
			if err != nil || done {
				return err
			}
		}
	}
}

func (r *Replay) replayBatch(ctx context.Context, publisher *broker.Publisher) (bool, error) {
	events, err := r.cfg.Store.publishedWithin(ctx, r.cfg.Since, r.cfg.Until, r.cfg.BatchSize)
	if err != nil || len(events) == 0 {
		return true, err
	}
	if _, err := publisher.PublishBatch(ctx, messages(events)); err != nil {
		return false, err
	}
	if err := markPublished(ctx, r.cfg.Store.pool, events); err != nil {
		return false, err
	}
	r.replayed += int64(len(events))
	return false, nil
}
