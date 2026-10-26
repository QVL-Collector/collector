package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"collector/sandbox/xp02/internal/broker"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/report"
)

const (
	defaultHost      = "localhost"
	defaultPort      = "5672"
	defaultVhost     = "collector"
	defaultPlatform  = "docker-compose"
	detachedDeadline = 5 * time.Second
)

var DefaultBackoff = broker.Backoff{Initial: 250 * time.Millisecond, Max: 10 * time.Second}

type AMQP struct {
	Endpoint broker.Endpoint
	User     string
}

func LoadAMQP(env *config.Env, binary string) AMQP {
	user := env.Required("XP02_AMQP_USER")
	password := env.Required("XP02_AMQP_PASSWORD")
	host := env.String("XP02_AMQP_HOST", defaultHost)
	port := env.String("XP02_AMQP_PORT", defaultPort)
	vhost := env.String("XP02_AMQP_VHOST", defaultVhost)
	target := url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + vhost,
	}
	return AMQP{
		Endpoint: broker.Endpoint{URL: target.String(), ConnectionName: binary + "@" + user},
		User:     user,
	}
}

func LoadLabels(env *config.Env) report.Labels {
	return report.Labels{
		Session:   env.String("XP02_SESSION", ""),
		Platform:  env.String("XP02_PLATEFORME", defaultPlatform),
		Criterion: env.String("XP02_CRITERE", ""),
	}
}

func Logger(binary string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("binaire", binary)
}

func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func Detached(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), detachedDeadline)
}

func OpenPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("configuration de PostgreSQL : %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connexion à PostgreSQL : %w", err)
	}
	return pool, nil
}

func Supervisor(settings AMQP, log *slog.Logger, rep *report.Reporter) broker.Supervisor {
	return broker.Supervisor{
		Endpoint: settings.Endpoint,
		Backoff:  DefaultBackoff,
		Log:      log,
		OnReconnect: func(reconnection broker.Reconnection) {
			rep.Seconds("reconnexion", reconnection.Downtime, fmt.Sprintf("%d tentatives de connexion échouées", reconnection.Attempts))
		},
	}
}

func ReportFault(rep *report.Reporter, err error) {
	if code, reason, ok := broker.Fault(err); ok {
		rep.Emit("refus_broker", strconv.Itoa(code), "code AMQP", reason)
	}
}
