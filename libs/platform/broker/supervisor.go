package broker

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	heartbeat = 10 * time.Second
	locale    = "en_US"
)

type Worker func(ctx context.Context, conn *amqp.Connection) error

type Supervisor struct {
	endpoint  Endpoint
	backoff   Backoff
	logger    *slog.Logger
	connected atomic.Bool
}

func NewSupervisor(endpoint Endpoint, logger *slog.Logger) *Supervisor {
	return &Supervisor{endpoint: endpoint, backoff: DefaultBackoff, logger: logger}
}

func (s *Supervisor) Ready(context.Context) error {
	if !s.connected.Load() {
		return ErrNotConnected
	}
	return nil
}

func (s *Supervisor) Run(ctx context.Context, work Worker) error {
	attempt := 0
	for ctx.Err() == nil {
		conn, err := s.dial()
		if err != nil {
			s.logger.WarnContext(ctx, "connexion au broker impossible",
				slog.String("event", "canal.connexion_impossible"),
				slog.String("connexion", s.endpoint.ConnectionName),
				slog.Int("tentative", attempt+1),
				slog.Any("erreur", err),
			)
			pause(ctx, s.backoff.Delay(attempt))
			attempt++
			continue
		}
		attempt = 0
		if err := s.serve(ctx, conn, work); err != nil {
			return err
		}
		pause(ctx, s.backoff.Delay(attempt))
	}
	return nil
}

func (s *Supervisor) serve(ctx context.Context, conn *amqp.Connection, work Worker) error {
	s.connected.Store(true)
	s.logger.InfoContext(ctx, "connecté au broker",
		slog.String("event", "canal.connecte"),
		slog.String("connexion", s.endpoint.ConnectionName),
	)
	err := work(ctx, conn)
	s.connected.Store(false)
	_ = conn.Close()
	if ctx.Err() != nil {
		return nil
	}
	if IsFault(err) {
		return fmt.Errorf("refus du broker sur %s : %w", s.endpoint.ConnectionName, err)
	}
	s.logger.WarnContext(ctx, "connexion au broker interrompue",
		slog.String("event", "canal.connexion_interrompue"),
		slog.String("connexion", s.endpoint.ConnectionName),
		slog.Any("erreur", err),
	)
	return nil
}

func (s *Supervisor) dial() (*amqp.Connection, error) {
	properties := amqp.NewConnectionProperties()
	properties.SetClientConnectionName(s.endpoint.ConnectionName)
	conn, err := amqp.DialConfig(s.endpoint.url, amqp.Config{
		Heartbeat:  heartbeat,
		Locale:     locale,
		Properties: properties,
	})
	if err != nil {
		return nil, fmt.Errorf("ouverture de la connexion %s : %w", s.endpoint.ConnectionName, err)
	}
	return conn, nil
}
