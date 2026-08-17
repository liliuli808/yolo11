package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_BUCKET", "uploads")
	t.Setenv("JWT_SIGNING_KEY", "secret")
	t.Setenv("EMAIL_CODE_KEY", "email-code-secret-key")
	t.Setenv("TURNSTILE_SECRET_KEY", "test-turnstile-secret")
	t.Setenv("EMAIL_FROM", "noreply@example.com")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8080")
}

func TestLoad_AllRequired(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("unexpected DatabaseURL: %q", cfg.DatabaseURL)
	}
	if cfg.ServerPort != "8081" {
		t.Errorf("expected default ServerPort 8081, got %q", cfg.ServerPort)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel info, got %q", cfg.LogLevel)
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
	if !containsMissingVar(err, "DATABASE_URL") {
		t.Errorf("error should mention DATABASE_URL: %v", err)
	}
}

func TestLoad_MissingRedisURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("REDIS_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing REDIS_URL")
	}
	if !containsMissingVar(err, "REDIS_URL") {
		t.Errorf("error should mention REDIS_URL: %v", err)
	}
}

func TestLoad_MissingS3Endpoint(t *testing.T) {
	setValidEnv(t)
	t.Setenv("S3_ENDPOINT", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing S3_ENDPOINT")
	}
	if !containsMissingVar(err, "S3_ENDPOINT") {
		t.Errorf("error should mention S3_ENDPOINT: %v", err)
	}
}

func TestLoad_MissingS3Bucket(t *testing.T) {
	setValidEnv(t)
	t.Setenv("S3_BUCKET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing S3_BUCKET")
	}
	if !containsMissingVar(err, "S3_BUCKET") {
		t.Errorf("error should mention S3_BUCKET: %v", err)
	}
}

func TestLoad_MissingJWTSigningKey(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_SIGNING_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing JWT_SIGNING_KEY")
	}
	if !containsMissingVar(err, "JWT_SIGNING_KEY") {
		t.Errorf("error should mention JWT_SIGNING_KEY: %v", err)
	}
}

func TestLoad_MissingEmailFrom(t *testing.T) {
	setValidEnv(t)
	t.Setenv("EMAIL_FROM", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing EMAIL_FROM")
	}
	if !containsMissingVar(err, "EMAIL_FROM") {
		t.Errorf("error should mention EMAIL_FROM: %v", err)
	}
}

func TestLoad_MissingPublicBaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PUBLIC_BASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing PUBLIC_BASE_URL")
	}
	if !containsMissingVar(err, "PUBLIC_BASE_URL") {
		t.Errorf("error should mention PUBLIC_BASE_URL: %v", err)
	}
}

func TestLoad_MissingTurnstileSecretKey(t *testing.T) {
	setValidEnv(t)
	t.Setenv("TURNSTILE_SECRET_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing TURNSTILE_SECRET_KEY")
	}
	if !containsMissingVar(err, "TURNSTILE_SECRET_KEY") {
		t.Errorf("error should mention TURNSTILE_SECRET_KEY: %v", err)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SERVER_PORT", "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ServerPort != "3000" {
		t.Errorf("expected ServerPort 3000, got %q", cfg.ServerPort)
	}
}

func TestLoad_EnvFileOverride(t *testing.T) {
	setValidEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, ".env.test")
	if err := os.WriteFile(path, []byte("SERVER_PORT=9090\n"), 0644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	cfg, err := Load(WithEnvFile(path))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ServerPort != "9090" {
		t.Errorf("expected env file to set SERVER_PORT=9090, got %q", cfg.ServerPort)
	}
}

func TestLoad_EnvFileDoesNotMutateProcessEnv(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SERVER_PORT", "1111")

	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("SERVER_PORT=2222\n"), 0644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	cfg, err := Load(WithEnvFile(path))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.ServerPort != "2222" {
		t.Errorf("expected env file override 2222, got %q", cfg.ServerPort)
	}
	if os.Getenv("SERVER_PORT") != "1111" {
		t.Errorf("process env was mutated: expected 1111, got %q", os.Getenv("SERVER_PORT"))
	}
}

func TestLoad_OptionalPoolSettings(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_MAX_CONNS", "50")
	t.Setenv("DATABASE_MIN_CONNS", "10")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "30m")
	t.Setenv("DATABASE_HEALTH_CHECK_PERIOD", "1m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseMaxConns != 50 {
		t.Errorf("expected DatabaseMaxConns 50, got %d", cfg.DatabaseMaxConns)
	}
	if cfg.DatabaseMinConns != 10 {
		t.Errorf("expected DatabaseMinConns 10, got %d", cfg.DatabaseMinConns)
	}
	if cfg.DatabaseMaxConnLifetime.Minutes() != 30 {
		t.Errorf("expected DatabaseMaxConnLifetime 30m, got %v", cfg.DatabaseMaxConnLifetime)
	}
	if cfg.DatabaseHealthCheckPeriod.Minutes() != 1 {
		t.Errorf("expected DatabaseHealthCheckPeriod 1m, got %v", cfg.DatabaseHealthCheckPeriod)
	}
}

func TestLoad_CORSAllowedOrigins(t *testing.T) {
	setValidEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, https://app.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Errorf("expected 2 CORS origins, got %d", len(cfg.CORSAllowedOrigins))
	}
}

func TestLoad_RateLimitBehindProxy(t *testing.T) {
	setValidEnv(t)
	t.Setenv("RATE_LIMIT_BEHIND_PROXY", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.RateLimitBehindProxy {
		t.Error("expected RateLimitBehindProxy to be true")
	}
}

func TestLoad_StaffUsernames(t *testing.T) {
	setValidEnv(t)
	t.Setenv("STAFF_USERNAMES", "a,b")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !reflect.DeepEqual(cfg.StaffUsernames, []string{"a", "b"}) {
		t.Errorf("expected StaffUsernames [a b], got %v", cfg.StaffUsernames)
	}
}

func TestLoad_TurnstileConfig(t *testing.T) {
	setValidEnv(t)
	t.Setenv("TURNSTILE_SITE_KEY", "test-site-key")
	t.Setenv("TURNSTILE_EXPECTED_HOSTNAME", "example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.TurnstileSecretKey != "test-turnstile-secret" {
		t.Errorf("unexpected TurnstileSecretKey: %q", cfg.TurnstileSecretKey)
	}
	if cfg.TurnstileSiteKey != "test-site-key" {
		t.Errorf("unexpected TurnstileSiteKey: %q", cfg.TurnstileSiteKey)
	}
	if cfg.TurnstileExpectedHostname != "example.com" {
		t.Errorf("unexpected TurnstileExpectedHostname: %q", cfg.TurnstileExpectedHostname)
	}
}

func containsMissingVar(err error, name string) bool {
	return err != nil && strings.Contains(err.Error(), name)
}
