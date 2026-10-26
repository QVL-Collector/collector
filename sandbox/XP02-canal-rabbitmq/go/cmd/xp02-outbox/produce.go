package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/sandbox/xp02/internal/app"
	"collector/sandbox/xp02/internal/config"
	"collector/sandbox/xp02/internal/decision"
	"collector/sandbox/xp02/internal/envelope"
	"collector/sandbox/xp02/internal/ident"
	"collector/sandbox/xp02/internal/outbox"
	"collector/sandbox/xp02/internal/report"
	"collector/sandbox/xp02/internal/stats"
)

const (
	initialRevision = 1
	insertDecision  = `
INSERT INTO xp02_decision_publication (decision_id, article_id, revision, issue, decided_at)
VALUES ($1::text::uuid, $2::text::uuid, $3, $4, $5)`
)

var issueCycle = []decision.Issue{decision.Published, decision.Refused, decision.ToVerify}

type produceSettings struct {
	labels     report.Labels
	exchange   string
	routingKey string
	count      int
}

func produce(ctx context.Context, log *slog.Logger) error {
	var env config.Env
	cfg := produceSettings{
		labels:     app.LoadLabels(&env),
		exchange:   env.String("XP02_EXCHANGE", defaultExchange),
		routingKey: env.String("XP02_ROUTING_KEY", defaultRoutingKey),
		count:      env.Int("XP02_COUNT", defaultCount),
	}
	env.Require(cfg.count > 0, "XP02_COUNT doit être strictement positif")
	if err := env.Err(); err != nil {
		return err
	}
	pool, err := openPool(ctx, &env)
	if err != nil {
		return err
	}
	defer pool.Close()
	correlationID, err := ident.New()
	if err != nil {
		return err
	}

	var latencies stats.Latencies
	started := time.Now()
	written := 0
	for sequence := range cfg.count {
		if ctx.Err() != nil {
			break
		}
		begin := time.Now()
		if err := recordDecision(ctx, pool, cfg, correlationID, sequence); err != nil {
			return err
		}
		latencies.Add(time.Since(begin))
		written++
	}
	elapsed := time.Since(started)
	log.Info("décisions écrites avec leur événement d'outbox", "nombre", written, "correlation_id", correlationID)

	rep := report.New(os.Stdout, cfg.labels)
	rep.Count("decisions_ecrites", int64(written), "transactions", "décision et événement d'outbox dans la même transaction")
	rep.Latencies("transaction_outbox", latencies.Summary())
	rep.Rate("debit_ecriture_outbox", int64(written), elapsed)
	return rep.Err()
}

func recordDecision(ctx context.Context, pool *pgxpool.Pool, cfg produceSettings, correlationID string, sequence int) error {
	articleID, err := ident.New()
	if err != nil {
		return err
	}
	decided, err := decision.New(articleID, initialRevision, issueCycle[sequence%len(issueCycle)], time.Now())
	if err != nil {
		return err
	}
	env, err := envelope.New(decision.EventType, decision.ContractVersion, correlationID, decided, decided.DecidedAt)
	if err != nil {
		return err
	}
	event, err := outbox.FromEnvelope(env, cfg.exchange, cfg.routingKey)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := saveDecision(ctx, tx, decided); err != nil {
			return err
		}
		return outbox.Enqueue(ctx, tx, event)
	})
}

func saveDecision(ctx context.Context, tx pgx.Tx, d decision.Decision) error {
	_, err := tx.Exec(ctx, insertDecision, d.ID, d.ArticleID, d.Revision, string(d.Issue), d.DecidedAt)
	if err != nil {
		return fmt.Errorf("écriture de la décision %s : %w", d.ID, err)
	}
	return nil
}
