package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"collector/libs/platform/metrics"
)

const (
	meterScope     = "collector/libs/platform/outbox"
	backlogTimeout = 2 * time.Second
)

const backlogQuery = `
SELECT count(*), COALESCE(EXTRACT(EPOCH FROM now() - min(cree_le)), 0)::float8
FROM outbox
WHERE publie_le IS NULL`

var relayRetries = metrics.Counter(otel.Meter(meterScope), "outbox.relances", "{essai}", "Cycles du relais différés après un lot non publié")

type backlog struct {
	pending          int64
	oldestAgeSeconds float64
}

func ObserveBacklog(pool *pgxpool.Pool) error {
	meter := otel.Meter(meterScope)
	pending, err := meter.Int64ObservableGauge("outbox.non_publies",
		metric.WithUnit("{evenement}"),
		metric.WithDescription("Lignes de l'outbox en attente de publication"),
	)
	if err != nil {
		return fmt.Errorf("jauge des lignes non publiées : %w", err)
	}
	oldestAge, err := meter.Float64ObservableGauge("outbox.plus_ancien_non_publie.age",
		metric.WithUnit("s"),
		metric.WithDescription("Âge de la plus ancienne ligne de l'outbox non publiée, 0 si aucune"),
	)
	if err != nil {
		return fmt.Errorf("jauge de l'âge de l'outbox : %w", err)
	}
	_, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		current, err := readBacklog(ctx, pool)
		if err != nil {
			return err
		}
		observer.ObserveInt64(pending, current.pending)
		observer.ObserveFloat64(oldestAge, current.oldestAgeSeconds)
		return nil
	}, pending, oldestAge)
	if err != nil {
		return fmt.Errorf("relevé de l'outbox : %w", err)
	}
	return nil
}

func readBacklog(ctx context.Context, pool *pgxpool.Pool) (backlog, error) {
	ctx, cancel := context.WithTimeout(ctx, backlogTimeout)
	defer cancel()
	var current backlog
	if err := pool.QueryRow(ctx, backlogQuery).Scan(&current.pending, &current.oldestAgeSeconds); err != nil {
		return backlog{}, fmt.Errorf("lecture de l'attente de l'outbox : %w", err)
	}
	return current, nil
}
