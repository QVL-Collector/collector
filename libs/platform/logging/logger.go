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

type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	if id, found := CorrelationID(ctx); found {
		record.AddAttrs(slog.String(correlationKey, id))
	}
	return h.Handler.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name)}
}
