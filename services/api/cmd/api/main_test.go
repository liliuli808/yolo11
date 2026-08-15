package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/platform/config"
	"github.com/yiguan/api/internal/platform/httpx"
)

func newTestApp(t *testing.T) *application {
	t.Helper()
	return &application{
		cfg: &config.Config{
			ServerPort:           "8081",
			LogLevel:             "info",
			RateLimitBehindProxy: false,
		},
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		limiter: &httpx.StubLimiter{},
	}
}

func newTestRouter(t *testing.T, app *application, authHandler *auth.Handler) http.Handler {
	t.Helper()
	return newRouter(app, authHandler, nil, nil, nil)
}

func TestHealthz_ReturnsOkOnly(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("expected exactly one field, got %d", len(body))
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestHealthz_IncludesRequestID(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestV1Healthz_ReturnsOkOnly(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if len(body) != 1 {
		t.Errorf("expected exactly one field, got %d", len(body))
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestV1Healthz_IncludesRequestID(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestReadyz_ReturnsUnavailableWithoutDependencies(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "unavailable" {
		t.Errorf("expected status unavailable, got %q", body["status"])
	}
	if body["db"] != "unavailable" {
		t.Errorf("expected db unavailable, got %q", body["db"])
	}
	if body["cache"] != "unavailable" {
		t.Errorf("expected cache unavailable, got %q", body["cache"])
	}
}

func TestReadyz_IncludesRequestID(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	router.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestNotFound_ReturnsStandardError(t *testing.T) {
	app := newTestApp(t)
	router := newRouter(app, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["code"] == "" {
		t.Error("expected error code in not found response")
	}
	if body["requestId"] == "" {
		t.Error("expected requestId in not found response")
	}
}

func TestV1Router_AppliesRateLimitMiddleware(t *testing.T) {
	limiter := &mockLimiter{allowed: false, retryAfter: 5 * time.Second}
	app := newTestApp(t)
	app.limiter = limiter

	v1 := newV1Router(app, nil, nil, nil, nil)
	// chi.Router implements http.Handler; mount a test route to exercise middleware.
	v1.(chi.Router).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when rate limited")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	v1.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
	if !limiter.called {
		t.Error("expected rate limiter to be invoked for v1 request")
	}
}

func TestV1Router_RateLimitKeyIncludesClientIP(t *testing.T) {
	limiter := &mockLimiter{allowed: true}
	app := newTestApp(t)
	app.cfg.RateLimitBehindProxy = true
	app.limiter = limiter

	v1 := newV1Router(app, nil, nil, nil, nil)
	v1.(chi.Router).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
	v1.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	expectedKey := "GET:/test:203.0.113.5"
	if limiter.lastKey != expectedKey {
		t.Errorf("expected rate limit key %q, got %q", expectedKey, limiter.lastKey)
	}
}

type mockLimiter struct {
	allowed    bool
	retryAfter time.Duration
	called     bool
	lastKey    string
}

func (m *mockLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	m.called = true
	m.lastKey = key
	return m.allowed, m.retryAfter, nil
}
