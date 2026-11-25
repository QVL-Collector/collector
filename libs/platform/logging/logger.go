package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	serviceKey     = "service"
	correlationKey = "correlation_id"
	traceKey       = "trace_id"
	spanKey        = "span_id"
	eventKey       = "event"
	auditKey       = "audit"
)

func New(service string, out io.Writer, level slog.Leveler) *slog.Logger {
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level, ReplaceAttr: maskSensitive})
	return slog.New(contextHandler{Handler: handler}).With(slog.String(serviceKey, service))
}

func ParseLevel(raw string) (slog.Level, error) {
	if strings.TrimSpace(raw) == "" {
		return slog.LevelInfo, nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo, fmt.Errorf("niveau de journal inconnu : %w", err)
	}
	return level, nil
}

func Audit(ctx context.Context, logger *slog.Logger, message, event string, attrs ...slog.Attr) {
	marked := append([]slog.Attr{slog.String(eventKey, event), slog.Bool(auditKey, true)}, attrs...)
	logger.LogAttrs(ctx, slog.LevelInfo, message, marked...)
}

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if id, found := CorrelationID(ctx); found {
		record.AddAttrs(slog.String(correlationKey, id))
	}
	if current, found := currentSpan(ctx); found {
		record.AddAttrs(slog.String(traceKey, current.TraceID().String()), slog.String(spanKey, current.SpanID().String()))
	}
	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}
