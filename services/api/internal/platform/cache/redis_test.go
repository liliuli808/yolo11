package cache

import (
	"context"
	"crypto/tls"
	"testing"
	"time"
)

func TestNewClient_InvalidAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewClient(ctx, "redis://invalid:abc")
	if err == nil {
		t.Fatal("expected error for invalid redis address")
	}
}

func TestNewClient_EmptyAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewClient(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty redis address")
	}
}

func TestParseRedisURL(t *testing.T) {
	opts, err := parseRedisURL("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Addr != "localhost:6379" {
		t.Errorf("expected addr localhost:6379, got %q", opts.Addr)
	}
	if opts.DB != 0 {
		t.Errorf("expected db 0, got %d", opts.DB)
	}
}

func TestParseRedisURL_RedissSetsServerName(t *testing.T) {
	opts, err := parseRedisURL("rediss://redis.example.com:6380/0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.TLSConfig == nil {
		t.Fatal("expected TLS config for rediss URL")
	}
	if opts.TLSConfig.ServerName != "redis.example.com" {
		t.Errorf("expected ServerName redis.example.com, got %q", opts.TLSConfig.ServerName)
	}
	if opts.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS 1.2, got %v", opts.TLSConfig.MinVersion)
	}
}

func TestParseRedisURL_RedissWithoutPort(t *testing.T) {
	opts, err := parseRedisURL("rediss://redis.example.com/0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.TLSConfig == nil || opts.TLSConfig.ServerName != "redis.example.com" {
		t.Errorf("expected ServerName redis.example.com, got %q", opts.TLSConfig.ServerName)
	}
}
