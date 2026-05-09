package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestLimiter_RespectsInterval(t *testing.T) {
	rpm := 60 // 1 per second
	limiter := NewLimiter(rpm)
	ctx := context.Background()

	// First request should not wait
	waited, err := limiter.Wait(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if waited != 0 {
		t.Fatalf("first request should not wait, got %s", waited)
	}

	// Second request should wait ~1 second
	start := time.Now()
	waited, err = limiter.Wait(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if waited < 900*time.Millisecond {
		t.Fatalf("expected ~1s wait, reported %s", waited)
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("expected ~1s elapsed, got %s", elapsed)
	}
}

func TestLimiter_NoLimit(t *testing.T) {
	limiter := NewLimiter(0)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		waited, err := limiter.Wait(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if waited != 0 {
			t.Fatalf("rpm=0 should never wait, got %s", waited)
		}
	}
}

func TestLimiter_NilSafe(t *testing.T) {
	var limiter *Limiter
	waited, err := limiter.Wait(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if waited != 0 {
		t.Fatalf("nil limiter should not wait, got %s", waited)
	}
}

func TestLimiter_ContextCancel(t *testing.T) {
	limiter := NewLimiter(1) // 1 RPM = 60s interval
	ctx := context.Background()

	// Burn the first free request
	limiter.Wait(ctx)

	// Cancel quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := limiter.Wait(ctx)
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
}

func TestLimiter_EvenSpacing(t *testing.T) {
	limiter := NewLimiter(120) // 2 per second = 500ms interval
	ctx := context.Background()

	var times []time.Time
	for i := 0; i < 3; i++ {
		limiter.Wait(ctx)
		times = append(times, time.Now())
	}

	for i := 1; i < len(times); i++ {
		gap := times[i].Sub(times[i-1])
		if gap < 450*time.Millisecond {
			t.Fatalf("gap between request %d and %d too short: %s", i-1, i, gap)
		}
	}
}
