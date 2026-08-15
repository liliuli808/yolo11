package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestID_AddsHeaderAndContext(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Error("expected request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get(RequestIDHeader) == "" {
		t.Error("expected response to include request ID header")
	}
}

func TestRequestID_PreservesExistingHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) != "existing-id" {
			t.Error("expected existing request ID to be preserved")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(RequestIDHeader, "existing-id")
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) != "existing-id" {
		t.Errorf("expected existing request ID to be returned, got %q", rec.Header().Get(RequestIDHeader))
	}
}

func TestLogger_LogsRequest(t *testing.T) {
	var buf strings.Builder
	handler := Logger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})), false)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	logOutput := buf.String()
	if !strings.Contains(logOutput, "method=POST") {
		t.Errorf("expected log to contain method, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "path=/test") {
		t.Errorf("expected log to contain path, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "status=201") {
		t.Errorf("expected log to contain status, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "client_ip=192.0.2.1") {
		t.Errorf("expected log to contain client_ip without port, got %q", logOutput)
	}
}

func TestLogger_LogsClientIPBehindProxy(t *testing.T) {
	var buf strings.Builder
	handler := Logger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})), true)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.2:8080")
	handler.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "client_ip=198.51.100.2") {
		t.Errorf("expected log to contain proxied client_ip without port, got %q", logOutput)
	}
}

func TestRecovery_RecoversPanic(t *testing.T) {
	var buf strings.Builder
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := Recovery(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", body.Code)
	}
	if body.Message != "internal server error" {
		t.Errorf("expected message internal server error, got %q", body.Message)
	}
	if body.RequestID == "" {
		t.Error("expected requestId in error response")
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Errorf("expected log to mention panic, got %q", buf.String())
	}
}

func TestRateLimit_AllowsWhenAllowed(t *testing.T) {
	limiter := &mockLimiter{allowed: true}
	handler := RateLimit(limiter, RateLimitConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !limiter.called {
		t.Error("expected rate limiter to be called")
	}
}

func TestRateLimit_BlocksWhenDenied(t *testing.T) {
	limiter := &mockLimiter{allowed: false, retryAfter: 5 * time.Second}
	handler := RateLimit(limiter, RateLimitConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when rate limited")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestClientIP_StripsPortFromRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:12345"

	ip := clientIP(r, false)
	if ip != "192.0.2.1" {
		t.Errorf("expected 192.0.2.1, got %q", ip)
	}
}

func TestClientIP_UsesRightmostXForwardedForBehindProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.2")

	ip := clientIP(r, true)
	if ip != "198.51.100.2" {
		t.Errorf("expected rightmost X-Forwarded-For address, got %q", ip)
	}
}

func TestClientIP_UsesSingleXForwardedForBehindProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.5")

	ip := clientIP(r, true)
	if ip != "203.0.113.5" {
		t.Errorf("expected single X-Forwarded-For address, got %q", ip)
	}
}

func TestClientIP_FallsBackToXRealIPBehindProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Real-Ip", "203.0.113.7")

	ip := clientIP(r, true)
	if ip != "203.0.113.7" {
		t.Errorf("expected X-Real-IP fallback, got %q", ip)
	}
}

func TestClientIP_IgnoresLeftmostSpoofedXForwardedForBehindProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.2")

	ip := clientIP(r, true)
	if ip == "1.2.3.4" {
		t.Errorf("leftmost spoofed X-Forwarded-For address must not be used")
	}
	if ip != "198.51.100.2" {
		t.Errorf("expected rightmost X-Forwarded-For address, got %q", ip)
	}
}

func TestClientIP_FallsBackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.2:12345"

	ip := clientIP(r, true)
	if ip != "192.0.2.2" {
		t.Errorf("expected RemoteAddr without port, got %q", ip)
	}
}

func TestClientIP_FallsBackWhenXForwardedForEndsWithComma(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.2:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.5,")

	ip := clientIP(r, true)
	if ip != "192.0.2.2" {
		t.Errorf("expected fallback to RemoteAddr when X-Forwarded-For ends with comma, got %q", ip)
	}
}

func TestClientIP_FallsBackWhenXForwardedForLastEntryEmpty(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.2:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, , 198.51.100.2,")

	ip := clientIP(r, true)
	if ip != "192.0.2.2" {
		t.Errorf("expected fallback to RemoteAddr when X-Forwarded-For last entry is empty, got %q", ip)
	}
}

func TestClientIP_StripsPortFromXForwardedFor(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.2:8080")

	ip := clientIP(r, true)
	if ip != "198.51.100.2" {
		t.Errorf("expected port stripped from X-Forwarded-For, got %q", ip)
	}
}

func TestClientIP_StripsPortFromXRealIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("X-Real-Ip", "203.0.113.7:9090")

	ip := clientIP(r, true)
	if ip != "203.0.113.7" {
		t.Errorf("expected port stripped from X-Real-IP, got %q", ip)
	}
}

func TestClientIP_FallsBackToXRealIPWhenXForwardedForWhitespaceOnly(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.2:12345"
	r.Header.Set("X-Forwarded-For", "   ")
	r.Header.Set("X-Real-Ip", "203.0.113.9")

	ip := clientIP(r, true)
	if ip != "203.0.113.9" {
		t.Errorf("expected fallback to X-Real-IP when X-Forwarded-For is whitespace, got %q", ip)
	}
}

type mockLimiter struct {
	allowed    bool
	retryAfter time.Duration
	called     bool
}

func (m *mockLimiter) Allow(ctx context.Context, key string) (bool, time.Duration, error) {
	m.called = true
	return m.allowed, m.retryAfter, nil
}
