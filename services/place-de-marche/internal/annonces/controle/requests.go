package controle

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/logging"
	"collector/libs/platform/message"
	"collector/libs/platform/outbox"
	"collector/services/place-de-marche/internal/annonces"
)

type RequestEncoder struct {
	validator *message.Validator
}

func NewRequestEncoder(validator *message.Validator) *RequestEncoder {
	return &RequestEncoder{validator: validator}
}

func (e *RequestEncoder) Encode(ctx context.Context, request annonces.ReviewRequest) (outbox.Event, error) {
	envelope, err := message.NewEnvelope(requestContract, correlationOf(ctx), toRequestData(request), time.Now())
	if err != nil {
		return outbox.Event{}, err
	}
	body, err := e.validator.Encode(envelope)
	if err != nil {
		return outbox.Event{}, fmt.Errorf("demande de contrôle de %s : %w", request.Article.ID, err)
	}
	return outbox.Event{
		ID:            envelope.ID,
		Exchange:      RequestExchange,
		RoutingKey:    routingKeyV1,
		Type:          envelope.Type,
		CorrelationID: envelope.CorrelationID,
		Body:          body,
		CreatedAt:     envelope.EmittedAt,
	}, nil
}

func correlationOf(ctx context.Context) string {
	if id, found := logging.CorrelationID(ctx); found {
		return id
	}
	return uuid.NewString()
}

func toRequestData(request annonces.ReviewRequest) requestData {
	article := request.Article
	return requestData{
		ArticleID:   article.ID.String(),
		Revision:    int(article.Revision),
		SubmittedAt: request.SubmittedAt.UTC(),
		Seller: sellerData{
			AccountID:   article.Seller.String(),
			MemberSince: request.Seller.MemberSince.Format(time.DateOnly),
			Suspended:   request.Seller.Suspended,
		},
		Listing: listingData{
			Title:        article.Title,
			Description:  article.Description,
			CategoryCode: categoryCode(article.Category),
			PriceCents:   priceCents(article.Price),
			PhotoCount:   request.PhotoCount,
		},
	}
}

func categoryCode(category *annonces.Category) *string {
	if category == nil {
		return nil
	}
	code := string(category.ID)
	return &code
}

func priceCents(price *annonces.Money) *int64 {
	if price == nil {
		return nil
	}
	cents := price.Cents
	return &cents
}
