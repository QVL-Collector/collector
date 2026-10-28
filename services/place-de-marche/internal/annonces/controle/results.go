package controle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"collector/libs/platform/broker"
	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/services/place-de-marche/internal/annonces"
)

var errUnexpectedContract = errors.New("contrat inattendu sur la file des résultats")

type Reviews interface {
	ApplyReview(ctx context.Context, outcome annonces.ReviewOutcome) (bool, error)
}

type ResultHandler struct {
	validator *message.Validator
	reviews   Reviews
	logger    *slog.Logger
}

func NewResultHandler(validator *message.Validator, reviews Reviews, logger *slog.Logger) *ResultHandler {
	return &ResultHandler{validator: validator, reviews: reviews, logger: logger}
}

func (h *ResultHandler) Handle(ctx context.Context, delivery broker.Delivery) (broker.Disposition, error) {
	var data resultData
	envelope, err := h.validator.Decode(delivery.Body, &data)
	if err != nil {
		return h.discard(ctx, delivery, err), nil
	}
	ctx = logging.WithCorrelationID(ctx, envelope.CorrelationID)
	if envelope.Type != resultContract.Type || envelope.Version != resultContract.Version {
		return h.discard(ctx, delivery, errUnexpectedContract), nil
	}
	outcome, err := toOutcome(envelope.ID, data)
	if err != nil {
		return h.discard(ctx, delivery, err), nil
	}
	applied, err := h.reviews.ApplyReview(ctx, outcome)
	if err != nil {
		return broker.Ack, fmt.Errorf("résultat %s : %w", envelope.ID, err)
	}
	h.logOutcome(ctx, outcome, applied)
	return broker.Ack, nil
}

func (h *ResultHandler) discard(ctx context.Context, delivery broker.Delivery, cause error) broker.Disposition {
	reason, _ := message.ReasonOf(cause)
	h.logger.WarnContext(ctx, "résultat de contrôle non conforme, écarté",
		slog.String("event", "controle.resultat_ecarte"),
		slog.String("message_id", delivery.MessageID),
		slog.String("raison", string(reason)),
		slog.Any("erreur", cause),
	)
	return broker.Reject
}

func (h *ResultHandler) logOutcome(ctx context.Context, outcome annonces.ReviewOutcome, applied bool) {
	if !applied {
		h.logger.InfoContext(ctx, "résultat de contrôle sans effet : doublon ou révision périmée",
			slog.String("event", "controle.resultat_ignore"),
			slog.String("message_id", outcome.MessageID),
			slog.String("article_id", outcome.Article.String()),
			slog.Int("revision", int(outcome.Revision)),
		)
		return
	}
	h.logger.InfoContext(ctx, "résultat de contrôle appliqué",
		slog.String("event", "controle.resultat_applique"),
		slog.String("message_id", outcome.MessageID),
		slog.String("article_id", outcome.Article.String()),
		slog.Int("revision", int(outcome.Revision)),
		slog.String("etat", string(outcome.Status)),
		slog.Int("version_regles", outcome.RulesVersion),
	)
}

func toOutcome(messageID string, data resultData) (annonces.ReviewOutcome, error) {
	id, err := annonces.ParseArticleID(data.ArticleID)
	if err != nil {
		return annonces.ReviewOutcome{}, err
	}
	status, err := annonces.ParseReviewIssue(data.Issue)
	if err != nil {
		return annonces.ReviewOutcome{}, err
	}
	reason, err := reviewReason(data.Reason)
	if err != nil {
		return annonces.ReviewOutcome{}, err
	}
	return annonces.ReviewOutcome{
		MessageID:    messageID,
		Article:      id,
		Revision:     annonces.Revision(data.Revision),
		Status:       status,
		Reason:       reason,
		RulesVersion: data.RulesVersion,
	}, nil
}

func reviewReason(raw *string) (*annonces.ReviewReason, error) {
	if raw == nil {
		return nil, nil
	}
	reason, err := annonces.ParseReviewReason(*raw)
	if err != nil {
		return nil, err
	}
	return &reason, nil
}
