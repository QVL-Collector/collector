package logging

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const CorrelationHeader = "X-Correlation-Id"

type correlationContextKey struct{}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationContextKey{}, id)
}

func WithReceivedCorrelationID(ctx context.Context, received string) context.Context {
	return WithCorrelationID(ctx, wellFormedOrNew(received))
}

func CorrelationID(ctx context.Context) (string, bool) {
	id, found := ctx.Value(correlationContextKey{}).(string)
	return id, found
}

func CorrelationIDOrNew(ctx context.Context) string {
	if id, found := CorrelationID(ctx); found {
		return id
	}
	return uuid.NewString()
}

func Resume(ctx context.Context, correlationID, traceParent string) context.Context {
	return ContinueTrace(WithReceivedCorrelationID(ctx, correlationID), traceParent)
}

func Restore(ctx context.Context, correlationID, traceParent string) context.Context {
	return restoreTrace(WithReceivedCorrelationID(ctx, correlationID), traceParent)
}

func StartOperation(ctx context.Context) context.Context {
	return ContinueTrace(WithCorrelationID(ctx, uuid.NewString()), "")
}

func Propagate(ctx context.Context, header http.Header) {
	if id, found := CorrelationID(ctx); found {
		header.Set(CorrelationHeader, id)
	}
	injectTraceParent(ctx, header)
}

func wellFormedOrNew(received string) string {
	parsed, err := uuid.Parse(received)
	if err != nil {
		return uuid.NewString()
	}
	return parsed.String()
}
