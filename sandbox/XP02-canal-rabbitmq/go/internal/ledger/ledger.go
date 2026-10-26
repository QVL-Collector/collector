package ledger

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	recordReception = `INSERT INTO xp02_reception (queue, event_id) VALUES ($1, $2::text::uuid) ON CONFLICT DO NOTHING`
	countReceptions = `SELECT count(*) FROM xp02_reception WHERE queue = $1`
)

type Memory struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewMemory() *Memory {
	return &Memory{seen: make(map[string]struct{})}
}

func (m *Memory) Record(_ context.Context, eventID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, known := m.seen[eventID]; known {
		return false, nil
	}
	m.seen[eventID] = struct{}{}
	return true, nil
}

func (m *Memory) Total(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.seen)), nil
}

type Postgres struct {
	pool  *pgxpool.Pool
	queue string
}

func NewPostgres(pool *pgxpool.Pool, queue string) *Postgres {
	return &Postgres{pool: pool, queue: queue}
}

func (p *Postgres) Record(ctx context.Context, eventID string) (bool, error) {
	tag, err := p.pool.Exec(ctx, recordReception, p.queue, eventID)
	if err != nil {
		return false, fmt.Errorf("enregistrement de la réception de %s : %w", eventID, err)
	}
	return tag.RowsAffected() == 1, nil
}

func (p *Postgres) Total(ctx context.Context) (int64, error) {
	var total int64
	if err := p.pool.QueryRow(ctx, countReceptions, p.queue).Scan(&total); err != nil {
		return 0, fmt.Errorf("comptage des réceptions de %s : %w", p.queue, err)
	}
	return total, nil
}
