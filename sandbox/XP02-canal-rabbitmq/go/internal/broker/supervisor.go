package broker

import (
	"context"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	heartbeat = 10 * time.Second
	locale    = "en_US"
)

type Endpoint struct {
	URL            string
	ConnectionName string
}

type Worker func(ctx context.Context, conn *amqp.Connection) error

type Reconnection struct {
	Downtime time.Duration
	Attempts int
}

type Supervisor struct {
	Endpoint    Endpoint
	Backoff     Backoff
	Log         *slog.Logger
	OnReconnect func(Reconnection)
}

func (s Supervisor) Run(ctx context.Context, work Worker) error {
	var lostAt time.Time
	attempt := 0
	for {
		conn, err := s.dial()
		if err != nil {
			s.Log.Warn("connexion au broker impossible", "tentative", attempt+1, "erreur", err)
			if !Pause(ctx, s.Backoff.Delay(attempt)) {
				return nil
			}
			attempt++
			continue
		}
		s.connected(lostAt, attempt)
		attempt = 0
		err = work(ctx, conn)
		_ = conn.Close()
		if err == nil || ctx.Err() != nil {
			return nil
		}
		if IsFault(err) {
			return err
		}
		s.Log.Warn("connexion au broker interrompue", "erreur", err)
		lostAt = time.Now()
	}
}

func (s Supervisor) dial() (*amqp.Connection, error) {
	properties := amqp.NewConnectionProperties()
	properties.SetClientConnectionName(s.Endpoint.ConnectionName)
	return amqp.DialConfig(s.Endpoint.URL, amqp.Config{
		Heartbeat:  heartbeat,
		Locale:     locale,
		Properties: properties,
	})
}

func (s Supervisor) connected(lostAt time.Time, attempts int) {
	if lostAt.IsZero() {
		s.Log.Info("connecté au broker", "connexion", s.Endpoint.ConnectionName)
		return
	}
	reconnection := Reconnection{Downtime: time.Since(lostAt), Attempts: attempts}
	s.Log.Info("reconnecté au broker", "interruption", reconnection.Downtime, "tentatives", attempts)
	if s.OnReconnect != nil {
		s.OnReconnect(reconnection)
	}
}
