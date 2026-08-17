package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimit defines a fixed-window limit: at most Count requests per Window.
type RateLimit struct {
	Count  int
	Window time.Duration
}

// RateLimiter decides whether a keyed operation is allowed within a limit.
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit RateLimit) (bool, time.Duration, error)
}

// MemoryLimiter is an in-process sliding-window rate limiter. It is suitable
// for a single API instance and can be replaced with a shared backend later.
type MemoryLimiter struct {
	mu      sync.Mutex
	entries map[string]*limitEntry
}

// NewMemoryLimiter creates a new MemoryLimiter.
func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{
		entries: make(map[string]*limitEntry),
	}
}

type limitEntry struct {
	timestamps []time.Time
}

// Allow returns true if key has not exceeded limit.Count events in the last
// limit.Window. When false, retryAfter is the time until the oldest event in
// the window expires.
func (m *MemoryLimiter) Allow(ctx context.Context, key string, limit RateLimit) (bool, time.Duration, error) {
	if limit.Count <= 0 {
		return true, 0, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	cutoff := now.Add(-limit.Window)

	entry, ok := m.entries[key]
	if !ok {
		entry = &limitEntry{}
		m.entries[key] = entry
	}

	// Drop events outside the window.
	pruned := entry.timestamps[:0]
	var oldest time.Time
	for _, ts := range entry.timestamps {
		if ts.After(cutoff) || ts.Equal(cutoff) {
			pruned = append(pruned, ts)
			if oldest.IsZero() || ts.Before(oldest) {
				oldest = ts
			}
		}
	}
	entry.timestamps = pruned

	if len(entry.timestamps) >= limit.Count {
		retryAfter := limit.Window - now.Sub(oldest)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter, nil
	}

	entry.timestamps = append(entry.timestamps, now)
	return true, 0, nil
}

// Reset removes all recorded events for a key. It is intended for tests.
func (m *MemoryLimiter) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, key)
}

// StubLimiter always allows requests. It can be used in tests that do not
// exercise rate limiting.
type StubLimiter struct{}

// Allow always returns true.
func (s *StubLimiter) Allow(ctx context.Context, key string, limit RateLimit) (bool, time.Duration, error) {
	return true, 0, nil
}

// Ensure the concrete types satisfy the interface at compile time.
var (
	_ RateLimiter = (*MemoryLimiter)(nil)
	_ RateLimiter = (*StubLimiter)(nil)
)

// keyBuilder helps construct consistent rate-limit keys.
type keyBuilder struct{}

func (keyBuilder) email(prefix, email string) string {
	return fmt.Sprintf("%s:email:%s", prefix, email)
}

func (keyBuilder) ip(prefix, ip string) string {
	return fmt.Sprintf("%s:ip:%s", prefix, ip)
}

func (keyBuilder) fingerprint(prefix, fp string) string {
	return fmt.Sprintf("%s:fp:%s", prefix, fp)
}

func (keyBuilder) username(prefix, username string) string {
	return fmt.Sprintf("%s:username:%s", prefix, username)
}
