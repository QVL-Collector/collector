package canal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"collector/libs/platform/broker"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/services/interactions/internal/notifications"
)

var errUnexpectedContract = errors.New("contrat inattendu sur la file des décisions de publication")

type Recorder interface {
	Record(ctx context.Context, decision notifications.Decision) (bool, error)
}

type DecisionHandler struct {
	validator *message.Validator
	recorder  Recorder
	logger    *slog.Logger
}

func NewDecisionHandler(validator *message.Validator, recorder Recorder, logger *slog.Logger) *DecisionHandler {
	return &DecisionHandler{validator: validator, recorder: recorder, logger: logger}
}

func (h *DecisionHandler) Handle(ctx context.Context, delivery broker.Delivery) (broker.Disposition, error) {
	var data decisionData
	envelope, err := h.validator.Decode(delivery.Body, &data)
	if err != nil {
		return h.discard(ctx, delivery, err), nil
	}
	ctx = logging.WithCorrelationID(ctx, envelope.CorrelationID)
	if envelope.Type != decisionContract.Type || envelope.Version != decisionContract.Version {
		return h.discard(ctx, delivery, errUnexpectedContract), nil
	}
	decision, err := toDecision(envelope.ID, data)
	if err != nil {
		return h.discard(ctx, delivery, err), nil
	}
	created, err := h.recorder.Record(ctx, decision)
	if err != nil {
		return broker.Ack, fmt.Errorf("décision de publication %s : %w", envelope.ID, err)
	}
	h.logRecord(ctx, decision, created)
	return broker.Ack, nil
}

func (h *DecisionHandler) discard(ctx context.Context, delivery broker.Delivery, cause error) broker.Disposition {
	reason, _ := message.ReasonOf(cause)
	h.logger.WarnContext(ctx, "décision de publication non conforme, écartée",
		slog.String("event", "notifications.decision_ecartee"),
		slog.String("message_id", delivery.MessageID),
		slog.String("raison", string(reason)),
		slog.Any("erreur", cause),
	)
	return broker.Reject
}

func (h *DecisionHandler) logRecord(ctx context.Context, decision notifications.Decision, created bool) {
	if !created {
		h.logger.InfoContext(ctx, "décision de publication déjà notifiée, sans effet",
			slog.String("event", "notifications.doublon_ignore"),
			slog.String("message_id", decision.MessageID),
			slog.String("article_id", decision.Article.String()),
			slog.Int("revision", int(decision.Revision)),
		)
		return
	}
	h.logger.InfoContext(ctx, "notification créée pour le vendeur",
		slog.String("event", "notifications.notification_creee"),
		slog.String("message_id", decision.MessageID),
		slog.String("article_id", decision.Article.String()),
		slog.Int("revision", int(decision.Revision)),
		slog.String("issue", string(decision.Issue)),
	)
}
