package app

import (
	"context"
	"sync/atomic"
	"time"
)

const idlePollInterval = time.Second

type Activity struct {
	last atomic.Int64
}

func (a *Activity) Touch() {
	a.last.Store(time.Now().UnixNano())
}

func (a *Activity) CancelWhenIdle(ctx context.Context, cancel context.CancelFunc, timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(idlePollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if a.idleFor(timeout) {
					cancel()
					return
				}
			}
		}
	}()
}

func (a *Activity) idleFor(timeout time.Duration) bool {
	last := a.last.Load()
	return last != 0 && time.Since(time.Unix(0, last)) > timeout
}
