package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Limiter enforces a maximum request rate with even spacing (no bursts).
type Limiter struct {
	mu       sync.Mutex
	interval time.Duration
	lastReq  time.Time
}

// NewLimiter creates a rate limiter. rpm = requests per minute. 0 = no limit.
func NewLimiter(rpm int) *Limiter {
	var interval time.Duration
	if rpm > 0 {
		interval = time.Minute / time.Duration(rpm)
	}
	return &Limiter{interval: interval}
}

// Wait blocks until the next request is allowed. Returns error if ctx is cancelled.
// Returns the duration waited (0 if no wait was needed).
func (l *Limiter) Wait(ctx context.Context) (time.Duration, error) {
	if l == nil || l.interval == 0 {
		return 0, nil
	}

	l.mu.Lock()
	wait := time.Until(l.lastReq.Add(l.interval))
	now := time.Now()
	if wait > 0 {
		l.lastReq = now.Add(wait)
	} else {
		l.lastReq = now
		wait = 0
	}
	l.mu.Unlock()

	if wait <= 0 {
		return 0, nil
	}

	fmt.Printf("[Rate Limit] waiting %s before request\n", wait.Round(time.Second))
	select {
	case <-time.After(wait):
		return wait, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}
