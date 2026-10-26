package stats

import (
	"math"
	"slices"
	"sync"
	"time"
)

type Summary struct {
	Count int
	P50   time.Duration
	P95   time.Duration
	P99   time.Duration
	Max   time.Duration
}

type Latencies struct {
	mu      sync.Mutex
	samples []time.Duration
}

func (l *Latencies) Add(samples ...time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.samples = append(l.samples, samples...)
}

func (l *Latencies) Summary() Summary {
	l.mu.Lock()
	sorted := slices.Clone(l.samples)
	l.mu.Unlock()
	if len(sorted) == 0 {
		return Summary{}
	}
	slices.Sort(sorted)
	return Summary{
		Count: len(sorted),
		P50:   percentile(sorted, 50),
		P95:   percentile(sorted, 95),
		P99:   percentile(sorted, 99),
		Max:   sorted[len(sorted)-1],
	}
}

func percentile(sorted []time.Duration, rank float64) time.Duration {
	index := int(math.Ceil(rank/100*float64(len(sorted)))) - 1
	return sorted[max(index, 0)]
}
