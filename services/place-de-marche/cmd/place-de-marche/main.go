package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"collector/libs/platform/logging"
)

const (
	serviceName      = "place-de-marche"
	serveCommand     = "serve"
	migrateCommand   = "migrate"
	suspendCommand   = "suspendre"
	reinstateCommand = "retablir"
	envLogLevel      = "JOURNAL_NIVEAU"
)

var errUnknownCommand = errors.New("commande inconnue (serve, migrate, suspendre <compte-id> ou retablir <compte-id> attendue)")

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(arguments []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()
	level, levelErr := logging.ParseLevel(os.Getenv(envLogLevel))
	logger := logging.New(serviceName, os.Stdout, level)
	if levelErr != nil {
		logger.WarnContext(ctx, "niveau de journal ignoré", slog.String("event", "demarrage.configuration"), slog.Any("erreur", levelErr))
	}
	command, operands := commandOf(arguments)
	if err := dispatch(ctx, logger, command, operands); err != nil {
		logger.ErrorContext(ctx, "arrêt sur erreur", slog.String("event", "arret.erreur"), slog.Any("erreur", err))
		return 1
	}
	return 0
}

func commandOf(arguments []string) (string, []string) {
	if len(arguments) == 0 {
		return serveCommand, nil
	}
	return arguments[0], arguments[1:]
}

func dispatch(ctx context.Context, logger *slog.Logger, command string, operands []string) error {
	switch command {
	case serveCommand:
		return serve(ctx, logger)
	case migrateCommand:
		return migrate(ctx, logger)
	case suspendCommand:
		return setSuspension(ctx, logger, operands, true)
	case reinstateCommand:
		return setSuspension(ctx, logger, operands, false)
	}
	return fmt.Errorf("%w : %q", errUnknownCommand, command)
}
