package main

import (
	"context"
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

const deliveryCountHeader = "x-delivery-count"

type handler struct {
	cfg         settings
	receptions  journal
	log         *slog.Logger
	activity    app.Activity
	latencies   stats.Latencies
	deliveries  int64
	redelivered int64
	foreign     int64
	invalid     int64
	duplicates  int64
	uniques     int64
	firstUnique time.Time
	lastUnique  time.Time
}

func newHandler(cfg settings, receptions journal, log *slog.Logger) *handler {
	return &handler{cfg: cfg, receptions: receptions, log: log}
}

func (h *handler) handle(ctx context.Context, delivery amqp.Delivery) (broker.Disposition, error) {
	h.activity.Touch()
	h.deliveries++
	if delivery.Redelivered {
		h.redelivered++
	}
	if delivery.UserId != h.cfg.expectedPublisher {
		h.foreign++
		h.log.Warn("émetteur inattendu, message écarté", "user_id", delivery.UserId, "message_id", delivery.MessageId)
		return broker.Reject, nil
	}
	env, err := envelope.Decode(delivery.Body, h.cfg.maxMessageBytes)
	if err != nil {
		h.invalid++
		h.log.Warn("message non conforme", "message_id", delivery.MessageId, "livraisons", delivery.Headers[deliveryCountHeader], "erreur", err)
		return h.cfg.invalidPolicy, nil
	}
	broker.Pause(ctx, h.cfg.processingDelay)
	first, err := h.receptions.Record(ctx, env.ID)
	if err != nil {
		return broker.Ack, fmt.Errorf("traitement de %s : %w", env.ID, err)
	}
	if !first {
		h.duplicates++
		return broker.Ack, nil
	}
	h.countUnique(env)
	return broker.Ack, nil
}

func (h *handler) countUnique(env envelope.Envelope) {
	now := time.Now()
	if h.uniques == 0 {
		h.firstUnique = now
	}
	h.uniques++
	h.lastUnique = now
	h.latencies.Add(now.Sub(env.EmittedAt))
}

func (h *handler) report(ctx context.Context, rep *report.Reporter) {
	queue := h.cfg.queue
	rep.Count("livraisons", h.deliveries, "messages", queue)
	rep.Count("uniques", h.uniques, "messages", queue)
	rep.Count("doublons", h.duplicates, "messages", queue)
	rep.Count("relivraisons", h.redelivered, "messages", queue)
	rep.Count("non_conformes", h.invalid, "messages", queue)
	rep.Count("emetteur_refuse", h.foreign, "messages", queue)
	rep.Latencies("latence_bout_en_bout", h.latencies.Summary())
	rep.Rate("debit_reception", h.uniques, h.lastUnique.Sub(h.firstUnique))
	h.reportTotal(ctx, rep)
}

func (h *handler) reportTotal(ctx context.Context, rep *report.Reporter) {
	detached, cancel := app.Detached(ctx)
	defer cancel()
	total, err := h.receptions.Total(detached)
	if err != nil {
		h.log.Error("comptage du journal de réception impossible", "erreur", err)
		return
	}
	rep.Count("uniques_cumules", total, "messages", h.cfg.queue+", journal "+h.cfg.ledger)
}
