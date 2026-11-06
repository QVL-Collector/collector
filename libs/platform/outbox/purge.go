package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	Retention     = 7 * 24 * time.Hour
	purgeInterval = time.Hour
	purgeQuery    = `SELECT purger_outbox_publiee()`
)

type Purger struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewPurger(pool *pgxpool.Pool, logger *slog.Logger) *Purger {
	return &Purger{pool: pool, logger: logger}
}

func (p *Purger) Run(ctx context.Context) error {
	ticker := time.NewTicker(purgeInterval)
	defer ticker.Stop()
	for {
		p.purge(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (p *Purger) purge(ctx context.Context) {
	var purged int64
	if err := p.pool.QueryRow(ctx, purgeQuery).Scan(&purged); err != nil {
		if ctx.Err() == nil {
			p.logger.WarnContext(ctx, "purge de l'outbox impossible",
				slog.String("event", "outbox.purge_en_echec"),
				slog.Any("erreur", err),
			)
		}
		return
	}
	if purged > 0 {
		p.logger.InfoContext(ctx, "lignes publiées de l'outbox purgées",
			slog.String("event", "outbox.purge"),
			slog.Int64("lignes", purged),
			slog.Duration("conservation", Retention),
		)
	}
}
