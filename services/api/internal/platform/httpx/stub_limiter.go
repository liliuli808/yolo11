package httpx

import (
	"context"
	"time"
)

// StubLimiter is a rate limiter that always allows requests. It is useful for
// wiring the RateLimit middleware before a production rate-limiting backend
// (e.g. Redis) is implemented.
type StubLimiter struct{}

// Allow always returns true with no retry delay.
func (s *StubLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	return true, 0, nil
}

// RateLimiterFunc adapts a function to the RateLimiter interface.
type RateLimiterFunc func(ctx context.Context, key string) (bool, time.Duration, error)

// Allow delegates to the wrapped function.
func (f RateLimiterFunc) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	return f(ctx, key)
}
