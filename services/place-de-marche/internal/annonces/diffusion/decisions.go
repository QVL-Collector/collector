package diffusion

import (
	"context"
	"fmt"
	"time"

	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/libs/platform/outbox"
	"collector/services/place-de-marche/internal/annonces"
)

type DecisionEncoder struct {
	validator *message.Validator
}

func NewDecisionEncoder(validator *message.Validator) *DecisionEncoder {
	return &DecisionEncoder{validator: validator}
}

func (e *DecisionEncoder) Encode(ctx context.Context, decision annonces.PublicationDecision) (outbox.Event, error) {
	envelope, err := message.NewEnvelope(decisionContract, logging.CorrelationIDOrNew(ctx), toDecisionData(decision), time.Now())
	if err != nil {
		return outbox.Event{}, err
	}
	body, err := e.validator.Encode(envelope)
	if err != nil {
		return outbox.Event{}, fmt.Errorf("décision de publication de %s : %w", decision.Article, err)
	}
	return outbox.NewEvent(envelope, EventExchange, decisionRoutingKey, body), nil
}

func toDecisionData(decision annonces.PublicationDecision) decisionData {
	return decisionData{
		ArticleID: decision.Article.String(),
		Revision:  int(decision.Revision),
		SellerID:  decision.Seller.String(),
		Issue:     string(decision.Outcome),
		Reason:    reasonCode(decision.Reason),
		DecidedAt: decision.DecidedAt.UTC(),
	}
}

func reasonCode(reason *annonces.ReviewReason) *string {
	if reason == nil {
		return nil
	}
	code := string(*reason)
	return &code
}
