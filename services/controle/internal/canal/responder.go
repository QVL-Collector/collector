package canal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"collector/libs/platform/broker"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/services/controle/internal/regles"
)

var errUnexpectedContract = errors.New("contrat inattendu sur la file des demandes")

type Responder struct {
	validator *message.Validator
	rules     regles.Rules
	logger    *slog.Logger
}

func NewResponder(validator *message.Validator, rules regles.Rules, logger *slog.Logger) *Responder {
	return &Responder{validator: validator, rules: rules, logger: logger}
}

func (r *Responder) HandlerFor(publisher *broker.Publisher) broker.Handler {
	return func(ctx context.Context, delivery broker.Delivery) (broker.Disposition, error) {
		return r.handle(ctx, delivery, publisher)
	}
}

func (r *Responder) handle(ctx context.Context, delivery broker.Delivery, publisher *broker.Publisher) (broker.Disposition, error) {
	var request requestData
	envelope, err := r.validator.Decode(delivery.Body, &request)
	if err != nil {
		return r.discard(ctx, delivery, err), nil
	}
	ctx = logging.WithCorrelationID(ctx, envelope.CorrelationID)
	if envelope.Type != requestContract.Type || envelope.Version != requestContract.Version {
		return r.discard(ctx, delivery, errUnexpectedContract), nil
	}
	verdict := r.rules.Evaluate(request.Listing.listing())
	result, err := r.resultMessage(request, verdict, envelope.CorrelationID)
	if err != nil {
		return r.discard(ctx, delivery, err), nil
	}
	if err := publisher.Publish(ctx, result); err != nil {
		return broker.Ack, fmt.Errorf("publication du résultat de %s : %w", envelope.ID, err)
	}
	r.logger.InfoContext(ctx, "demande contrôlée",
		slog.String("event", "controle.demande_controlee"),
		slog.String("message_id", envelope.ID),
		slog.String("article_id", request.ArticleID),
		slog.Int("revision", request.Revision),
		slog.String("issue", string(verdict.Issue)),
		slog.String("motif", string(verdict.Reason)),
		slog.Int("version_regles", regles.RulesVersion),
	)
	return broker.Ack, nil
}

func (r *Responder) resultMessage(request requestData, verdict regles.Verdict, correlationID string) (broker.Message, error) {
	envelope, err := message.NewEnvelope(resultContract, correlationID, toResultData(request, verdict), time.Now())
	if err != nil {
		return broker.Message{}, err
	}
	body, err := r.validator.Encode(envelope)
	if err != nil {
		return broker.Message{}, err
	}
	return broker.Message{
		Exchange:      resultExchange,
		RoutingKey:    routingKeyV1,
		ID:            envelope.ID,
		Type:          envelope.Type,
		CorrelationID: envelope.CorrelationID,
		Timestamp:     envelope.EmittedAt,
		Body:          body,
	}, nil
}

func (r *Responder) discard(ctx context.Context, delivery broker.Delivery, cause error) broker.Disposition {
	reason, _ := message.ReasonOf(cause)
	r.logger.WarnContext(ctx, "demande de contrôle non conforme, écartée",
		slog.String("event", "controle.demande_ecartee"),
		slog.String("message_id", delivery.MessageID),
		slog.String("raison", string(reason)),
		slog.Any("erreur", cause),
	)
	return broker.Reject
}
