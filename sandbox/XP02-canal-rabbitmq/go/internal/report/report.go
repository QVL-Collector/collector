package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"collector/sandbox/xp02/internal/stats"
)

const experiment = "XP02"

type Labels struct {
	Session   string
	Platform  string
	Criterion string
}

type Reporter struct {
	mu     sync.Mutex
	out    *csv.Writer
	labels Labels
	err    error
}

func New(w io.Writer, labels Labels) *Reporter {
	out := csv.NewWriter(w)
	out.Comma = ';'
	return &Reporter{out: out, labels: labels}
}

func (r *Reporter) Emit(step, value, unit, comment string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return
	}
	record := []string{
		time.Now().UTC().Format(time.RFC3339),
		experiment,
		r.labels.Session,
		r.labels.Platform,
		r.labels.Criterion,
		step,
		value,
		unit,
		"",
		"",
		comment,
	}
	if err := r.out.Write(record); err != nil {
		r.err = fmt.Errorf("écriture de la mesure %s : %w", step, err)
		return
	}
	r.out.Flush()
	r.err = r.out.Error()
}

func (r *Reporter) Count(step string, value int64, unit, comment string) {
	r.Emit(step, strconv.FormatInt(value, 10), unit, comment)
}

func (r *Reporter) Seconds(step string, d time.Duration, comment string) {
	r.Emit(step, decimal(d.Seconds()), "s", comment)
}

func (r *Reporter) Latencies(step string, summary stats.Summary) {
	comment := fmt.Sprintf("%d échantillons", summary.Count)
	quantiles := []struct {
		name  string
		value time.Duration
	}{
		{"p50", summary.P50},
		{"p95", summary.P95},
		{"p99", summary.P99},
		{"max", summary.Max},
	}
	for _, quantile := range quantiles {
		r.Emit(step+"_"+quantile.name, decimal(milliseconds(quantile.value)), "ms", comment)
	}
}

func (r *Reporter) Rate(step string, count int64, elapsed time.Duration) {
	if elapsed <= 0 {
		return
	}
	comment := fmt.Sprintf("%d messages en %s", count, elapsed.Round(time.Millisecond))
	r.Emit(step, decimal(float64(count)/elapsed.Seconds()), "msg/s", comment)
}

func (r *Reporter) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func milliseconds(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func decimal(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}
