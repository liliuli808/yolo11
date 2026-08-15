package httpx

import (
	"context"
	"testing"
	"time"
)

func TestStubLimiter_AllowsAllRequests(t *testing.T) {
	limiter := &StubLimiter{}

	allowed, retryAfter, err := limiter.Allow(context.Background(), "any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected stub limiter to allow request")
	}
	if retryAfter != 0 {
		t.Errorf("expected zero retry after, got %v", retryAfter)
	}
}

func TestStubLimiter_AllowsRequestsIndependently(t *testing.T) {
	limiter := &StubLimiter{}

	for i := 0; i < 5; i++ {
		allowed, _, err := limiter.Allow(context.Background(), "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("expected stub limiter to allow request %d", i)
		}
	}
}

var _ RateLimiter = (&StubLimiter{})
var _ RateLimiter = (RateLimiterFunc)(nil)

func TestRateLimiterFunc_Allows(t *testing.T) {
	limiter := RateLimiterFunc(func(ctx context.Context, key string) (bool, time.Duration, error) {
		return true, 0, nil
	})

	allowed, _, err := limiter.Allow(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected func limiter to allow")
	}
}
