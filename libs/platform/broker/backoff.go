package broker

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"
)

const maxBackoffExponent = 20

var DefaultBackoff = Backoff{Initial: 250 * time.Millisecond, Max: 10 * time.Second}

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
	return half + jitter(half)
}

func jitter(limit time.Duration) time.Duration {
	drawn, err := rand.Int(rand.Reader, big.NewInt(int64(limit)+1))
	if err != nil {
		return 0
	}
	return time.Duration(drawn.Int64())
}

func pause(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
