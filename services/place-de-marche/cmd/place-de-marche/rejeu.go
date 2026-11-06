package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"collector/libs/platform/outbox"
	"collector/services/place-de-marche/internal/annonces/stockage"
	"collector/services/place-de-marche/internal/config"
)

const (
	fromFlag  = "depuis"
	untilFlag = "jusqu-a"
)

var errReplayUsage = errors.New("usage : rejouer --depuis <RFC3339> [--jusqu-a <RFC3339>]")

func replayOutbox(ctx context.Context, logger *slog.Logger, operands []string) error {
	window, err := replayWindowOf(operands, time.Now())
	if err != nil {
		return err
	}
	database, err := config.LoadDatabase(os.Getenv)
	if err != nil {
		return err
	}
	pool, err := stockage.Connect(ctx, database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()
	replayed, err := outbox.Replay(ctx, pool, window)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "outbox rejouée",
		slog.String("event", "outbox.rejeu"),
		slog.String("correlation_id", uuid.NewString()),
		slog.Time("depuis", window.From),
		slog.Time("jusqu_a", window.To),
		slog.Int64("lignes", replayed),
	)
	return nil
}

func replayWindowOf(operands []string, now time.Time) (outbox.Window, error) {
	flags := flag.NewFlagSet(replayCommand, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	from := flags.String(fromFlag, "", "")
	until := flags.String(untilFlag, "", "")
	if err := flags.Parse(operands); err != nil || flags.NArg() > 0 || *from == "" {
		return outbox.Window{}, errReplayUsage
	}
	start, err := parseInstant(fromFlag, *from)
	if err != nil {
		return outbox.Window{}, err
	}
	end := now
	if *until != "" {
		if end, err = parseInstant(untilFlag, *until); err != nil {
			return outbox.Window{}, err
		}
	}
	return outbox.NewWindow(start, end, now)
}

func parseInstant(name, raw string) (time.Time, error) {
	instant, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("--%s : date RFC3339 attendue : %w", name, errReplayUsage)
	}
	return instant, nil
}
