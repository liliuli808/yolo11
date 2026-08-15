package database

import (
	"context"
	"testing"
	"time"
)

func TestNewPool_InvalidConnectionString(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewPool(ctx, "not-a-valid-url", PoolConfig{})
	if err == nil {
		t.Fatal("expected error for invalid connection string")
	}
}

func TestNewPool_EmptyConnectionString(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewPool(ctx, "", PoolConfig{})
	if err == nil {
		t.Fatal("expected error for empty connection string")
	}
}
