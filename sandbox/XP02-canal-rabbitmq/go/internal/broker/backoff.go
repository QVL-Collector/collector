package broker

import (
	"context"
	"math/rand/v2"
	"time"
)

const maxBackoffExponent = 20

type Backoff struct {
	Initial time.Duration
	Max     time.Duration
}

func (b Backoff) Delay(attempt int) time.Duration {
	ceiling := b.Initial << min(max(attempt, 0), maxBackoffExponent)
	if ceiling <= 0 || ceiling > b.Max {
		ceiling = b.Max
	}
	half := ceiling / 2
	return half + rand.N(half+1)
}

func Pause(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
