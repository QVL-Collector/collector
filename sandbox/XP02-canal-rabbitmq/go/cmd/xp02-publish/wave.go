package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/broker"
	"collector/sandbox/xp02/internal/envelope"
	"collector/sandbox/xp02/internal/report"
	"collector/sandbox/xp02/internal/stats"
)

const (
	contractVersion  = 1
	poisonedMessages = 2
)

type probe struct {
	Sequence int `json:"sequence"`
}

type batch struct {
	messages []broker.Message
	regular  bool
}

type wave struct {
	cfg           settings
	correlationID string
	log           *slog.Logger
	startedAt     time.Time
	finishedAt    time.Time
	produced      int
	confirmed     int
	rejected      int
	returned      int64
	poisoned      bool
	pending       *batch
	latencies     stats.Latencies
}

func newWave(cfg settings, correlationID string, log *slog.Logger) *wave {
	return &wave{cfg: cfg, correlationID: correlationID, log: log, startedAt: time.Now()}
}

func (w *wave) worker(ctx context.Context, conn *amqp.Connection) error {
	publisher, err := broker.OpenPublisher(conn, w.cfg.userID, w.cfg.confirmTimeout)
	if err != nil {
		return err
	}
	defer w.release(publisher)
	for !w.complete() {
		if err := w.publishNext(ctx, publisher); err != nil {
			return err
		}
	}
	w.finishedAt = time.Now()
	return nil
}

func (w *wave) release(publisher *broker.Publisher) {
	w.returned += publisher.Returned()
	publisher.Close()
}

func (w *wave) complete() bool {
	return w.confirmed >= w.cfg.count && (w.cfg.poisonAt == noPoison || w.poisoned)
}

func (w *wave) publishNext(ctx context.Context, publisher *broker.Publisher) error {
	if w.pending == nil {
		if !w.pace(ctx) {
			return ctx.Err()
		}
		next, err := w.nextBatch()
		if err != nil {
			return err
		}
		w.pending = &next
	}
	latencies, err := publisher.PublishBatch(ctx, w.pending.messages)
	if errors.Is(err, broker.ErrRejected) {
		return w.backOffAfterRejection(ctx, err)
	}
	if err != nil {
		return err
	}
	w.settle(latencies)
	return nil
}

func (w *wave) backOffAfterRejection(ctx context.Context, cause error) error {
	w.log.Warn("lot refusé par le broker, nouvel essai du même lot", "erreur", cause)
	w.rejected++
	if !broker.Pause(ctx, app.DefaultBackoff.Delay(w.rejected)) {
		return ctx.Err()
	}
	return nil
}

func (w *wave) settle(latencies []time.Duration) {
	if w.pending.regular {
		w.confirmed += len(latencies)
		w.latencies.Add(latencies...)
	} else {
		w.poisoned = true
	}
	w.pending = nil
}

func (w *wave) pace(ctx context.Context) bool {
	if w.cfg.rate == 0 {
		return true
	}
	due := w.startedAt.Add(time.Duration(w.produced) * time.Second / time.Duration(w.cfg.rate))
	return broker.Pause(ctx, time.Until(due))
}

func (w *wave) nextBatch() (batch, error) {
	if w.poisonDue() {
		return w.poisonBatch()
	}
	size := min(w.cfg.batchSize, w.cfg.count-w.produced)
	if w.cfg.poisonAt > w.produced && !w.poisoned {
		size = min(size, w.cfg.poisonAt-w.produced)
	}
	messages := make([]broker.Message, 0, size)
	for range size {
		message, err := w.message(w.produced)
		if err != nil {
			return batch{}, err
		}
		messages = append(messages, message)
		w.produced++
	}
	return batch{messages: messages, regular: true}, nil
}

func (w *wave) poisonDue() bool {
	return w.cfg.poisonAt != noPoison && !w.poisoned && w.produced >= w.cfg.poisonAt
}

func (w *wave) poisonBatch() (batch, error) {
	malformed, err := w.message(w.cfg.poisonAt)
	if err != nil {
		return batch{}, err
	}
	malformed.Body = malformed.Body[:len(malformed.Body)/2]
	duplicated, err := w.message(w.cfg.poisonAt)
	if err != nil {
		return batch{}, err
	}
	duplicated.Body = withDuplicatedID(duplicated)
	return batch{messages: []broker.Message{malformed, duplicated}}, nil
}

func withDuplicatedID(message broker.Message) []byte {
	prefix := fmt.Sprintf(`{"id":%q,`, message.ID)
	return append([]byte(prefix), message.Body[1:]...)
}

func (w *wave) message(sequence int) (broker.Message, error) {
	env, err := envelope.New(w.cfg.eventType, contractVersion, w.correlationID, probe{Sequence: sequence}, time.Now())
	if err != nil {
		return broker.Message{}, err
	}
	body, err := env.Encode()
	if err != nil {
		return broker.Message{}, err
	}
	return broker.Message{
		Exchange:      w.cfg.exchange,
		RoutingKey:    w.cfg.routingKey,
		ID:            env.ID,
		Type:          env.Type,
		CorrelationID: env.CorrelationID,
		Timestamp:     env.EmittedAt,
		Body:          body,
	}, nil
}

func (w *wave) report(rep *report.Reporter) {
	target := fmt.Sprintf("%s / %s", w.cfg.exchange, w.cfg.routingKey)
	rep.Count("demandes", int64(w.cfg.count), "messages", target)
	rep.Count("publies_confirmes", int64(w.confirmed), "messages", target)
	rep.Count("lots_refuses", int64(w.rejected), "lots", "nack du broker, lot republié")
	rep.Count("retournes_non_routes", w.returned, "messages", "basic.return, mandatory")
	if w.poisoned {
		rep.Count("pieges_publies", poisonedMessages, "messages", fmt.Sprintf("JSON tronqué et clé dupliquée après le message %d", w.cfg.poisonAt))
	}
	rep.Latencies("confirmation", w.latencies.Summary())
	rep.Rate("debit_publication", int64(w.confirmed), w.end().Sub(w.startedAt))
}

func (w *wave) end() time.Time {
	if w.finishedAt.IsZero() {
		return time.Now()
	}
	return w.finishedAt
}
