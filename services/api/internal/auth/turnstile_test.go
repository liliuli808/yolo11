package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudflareTurnstileVerifier_EmptyToken(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	v := &CloudflareTurnstileVerifier{
		secretKey:     "test-secret",
		client:        server.Client(),
		siteverifyURL: server.URL,
	}

	if err := v.Verify(context.Background(), ""); !errors.Is(err, ErrCaptchaFailed) {
		t.Fatalf("expected ErrCaptchaFailed, got %v", err)
	}
	if called {
		t.Fatal("expected no HTTP call for empty token")
	}
}

func TestCloudflareTurnstileVerifier_Success(t *testing.T) {
	var got struct {
		Secret   string `json:"secret"`
		Response string `json:"response"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json content type, got %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"hostname":"example.com"}`))
	}))
	defer server.Close()

	v := &CloudflareTurnstileVerifier{
		secretKey:     "test-secret",
		hostname:      "example.com",
		client:        server.Client(),
		siteverifyURL: server.URL,
	}

	if err := v.Verify(context.Background(), "test-token"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Secret != "test-secret" {
		t.Errorf("expected secret %q, got %q", "test-secret", got.Secret)
	}
	if got.Response != "test-token" {
		t.Errorf("expected response %q, got %q", "test-token", got.Response)
	}
}

func TestCloudflareTurnstileVerifier_ResponseFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer server.Close()

	v := &CloudflareTurnstileVerifier{
		secretKey:     "test-secret",
		client:        server.Client(),
		siteverifyURL: server.URL,
	}

	if err := v.Verify(context.Background(), "test-token"); !errors.Is(err, ErrCaptchaFailed) {
		t.Fatalf("expected ErrCaptchaFailed, got %v", err)
	}
}

func TestCloudflareTurnstileVerifier_HostnameMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"hostname":"other.com"}`))
	}))
	defer server.Close()

	v := &CloudflareTurnstileVerifier{
		secretKey:     "test-secret",
		hostname:      "example.com",
		client:        server.Client(),
		siteverifyURL: server.URL,
	}

	if err := v.Verify(context.Background(), "test-token"); !errors.Is(err, ErrCaptchaFailed) {
		t.Fatalf("expected ErrCaptchaFailed on hostname mismatch, got %v", err)
	}
}

func TestCloudflareTurnstileVerifier_HostnameCheckSkippedWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"hostname":"other.com"}`))
	}))
	defer server.Close()

	v := &CloudflareTurnstileVerifier{
		secretKey:     "test-secret",
		client:        server.Client(),
		siteverifyURL: server.URL,
	}

	if err := v.Verify(context.Background(), "test-token"); err != nil {
		t.Fatalf("verify with empty expected hostname: %v", err)
	}
}
