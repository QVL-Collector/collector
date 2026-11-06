package ratelimit

import (
	"math"
	"sync"
	"time"
)

const (
	maxTrackedKeys         = 50_000
	sweepInterval          = time.Minute
	maxRetryAfterInSeconds = 3600
)

type Clock func() time.Time

type bucket struct {
	tokens  float64
	updated time.Time
	limit   Limit
}

func (b *bucket) refill(now time.Time, limit Limit) {
	if elapsed := now.Sub(b.updated); elapsed > 0 {
		b.tokens += elapsed.Seconds() * limit.Rate
		b.updated = now
	}
	b.tokens = min(b.tokens, limit.capacity())
	b.limit = limit
}

func (b *bucket) full() bool {
	return b.tokens >= b.limit.capacity()
}

func (b *bucket) secondsUntilToken() int {
	wait := math.Ceil((1 - b.tokens) / b.limit.Rate)
	return int(math.Min(math.Max(wait, 1), maxRetryAfterInSeconds))
}

type demand struct {
	scope string
	key   string
	limit Limit
}

type refusal struct {
	scope             string
	retryAfterSeconds int
}

type store struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	clock     Clock
	maxKeys   int
	nextSweep time.Time
}

func newStore(clock Clock, maxKeys int) *store {
	return &store{buckets: make(map[string]*bucket), clock: clock, maxKeys: maxKeys}
}

func (s *store) admit(demands []demand) (refusal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock()
	s.sweepIfDue(now)
	granted := make([]*bucket, 0, len(demands))
	var refused refusal
	for _, d := range demands {
		b := s.bucketFor(d.key, d.limit, now)
		b.refill(now, d.limit)
		granted = append(granted, b)
		if b.tokens >= 1 {
			continue
		}
		if wait := b.secondsUntilToken(); wait > refused.retryAfterSeconds {
			refused = refusal{scope: d.scope, retryAfterSeconds: wait}
		}
	}
	if refused.retryAfterSeconds > 0 {
		return refused, false
	}
	for _, b := range granted {
		b.tokens--
	}
	return refusal{}, true
}

func (s *store) bucketFor(key string, limit Limit, now time.Time) *bucket {
	if existing, found := s.buckets[key]; found {
		return existing
	}
	if len(s.buckets) >= s.maxKeys {
		s.evictOne()
	}
	created := &bucket{tokens: limit.capacity(), updated: now, limit: limit}
	s.buckets[key] = created
	return created
}

func (s *store) evictOne() {
	for key := range s.buckets {
		delete(s.buckets, key)
		return
	}
}

func (s *store) sweepIfDue(now time.Time) {
	if now.Before(s.nextSweep) {
		return
	}
	for key, b := range s.buckets {
		b.refill(now, b.limit)
		if b.full() {
			delete(s.buckets, key)
		}
	}
	s.nextSweep = now.Add(sweepInterval)
}
