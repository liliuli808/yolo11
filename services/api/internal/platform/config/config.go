package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds validated application configuration loaded from the environment.
type Config struct {
	DatabaseURL               string
	RedisURL                  string
	S3Endpoint                string
	S3Bucket                  string
	JWTSigningKey             string
	EmailFrom                 string
	PublicBaseURL             string
	ServerPort                string
	LogLevel                  string
	Environment               string
	CORSAllowedOrigins        []string
	RateLimitBehindProxy      bool
	DatabaseMaxConns          int32
	DatabaseMinConns          int32
	DatabaseMaxConnLifetime   time.Duration
	DatabaseHealthCheckPeriod time.Duration

	EmailCodeKey    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	EmailAdapter    string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPTLS         bool
	StaffEmails     []string
	AdminSessionTTL time.Duration
}

// loaderState carries per-load overrides (e.g. from env files) without
// mutating the global process environment.
type loaderState struct {
	overrides map[string]string
}

// LoaderOption customizes how configuration is loaded.
type LoaderOption func(*loaderState) error

// WithEnvFile loads key=value pairs from the given file into the loader's
// private override map. These values take precedence over process environment
// variables for the current Load call only.
func WithEnvFile(path string) LoaderOption {
	return func(state *loaderState) error {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open env file: %w", err)
		}
		defer f.Close()

		if state.overrides == nil {
			state.overrides = make(map[string]string)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key == "" {
				continue
			}
			// Remove surrounding quotes if present.
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}
			state.overrides[key] = value
		}
		return scanner.Err()
	}
}

func (s *loaderState) getenv(key string) string {
	if s.overrides == nil {
		return os.Getenv(key)
	}
	if v, ok := s.overrides[key]; ok {
		return v
	}
	return os.Getenv(key)
}

// Load reads configuration from environment variables and validates that all
// required values are present. Empty strings are treated as missing.
func Load(opts ...LoaderOption) (*Config, error) {
	state := &loaderState{}
	for _, opt := range opts {
		if err := opt(state); err != nil {
			return nil, fmt.Errorf("apply loader option: %w", err)
		}
	}

	required := []struct {
		name  string
		value string
	}{
		{"DATABASE_URL", state.getenv("DATABASE_URL")},
		{"REDIS_URL", state.getenv("REDIS_URL")},
		{"S3_ENDPOINT", state.getenv("S3_ENDPOINT")},
		{"S3_BUCKET", state.getenv("S3_BUCKET")},
		{"JWT_SIGNING_KEY", state.getenv("JWT_SIGNING_KEY")},
		{"EMAIL_CODE_KEY", state.getenv("EMAIL_CODE_KEY")},
		{"EMAIL_FROM", state.getenv("EMAIL_FROM")},
		{"PUBLIC_BASE_URL", state.getenv("PUBLIC_BASE_URL")},
	}

	var missing []string
	for _, r := range required {
		if strings.TrimSpace(r.value) == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrMissingConfig, strings.Join(missing, ", "))
	}

	cfg := &Config{
		DatabaseURL:               required[0].value,
		RedisURL:                  required[1].value,
		S3Endpoint:                required[2].value,
		S3Bucket:                  required[3].value,
		JWTSigningKey:             required[4].value,
		EmailCodeKey:              required[5].value,
		EmailFrom:                 required[6].value,
		PublicBaseURL:             required[7].value,
		ServerPort:                defaultString(state.getenv("SERVER_PORT"), "8081"),
		LogLevel:                  defaultString(state.getenv("LOG_LEVEL"), "info"),
		Environment:               defaultString(state.getenv("ENVIRONMENT"), "development"),
		CORSAllowedOrigins:        parseStringList(state.getenv("CORS_ALLOWED_ORIGINS")),
		RateLimitBehindProxy:      parseBool(state.getenv("RATE_LIMIT_BEHIND_PROXY"), false),
		DatabaseMaxConns:          parseInt32(state.getenv("DATABASE_MAX_CONNS"), 25),
		DatabaseMinConns:          parseInt32(state.getenv("DATABASE_MIN_CONNS"), 5),
		DatabaseMaxConnLifetime:   parseDuration(state.getenv("DATABASE_MAX_CONN_LIFETIME"), time.Hour),
		DatabaseHealthCheckPeriod: parseDuration(state.getenv("DATABASE_HEALTH_CHECK_PERIOD"), 5*time.Minute),
		AccessTokenTTL:            parseDuration(state.getenv("ACCESS_TOKEN_TTL"), 15*time.Minute),
		RefreshTokenTTL:           parseDuration(state.getenv("REFRESH_TOKEN_TTL"), 7*24*time.Hour),
		EmailAdapter:              defaultString(state.getenv("EMAIL_ADAPTER"), "console"),
		SMTPHost:                  defaultString(state.getenv("SMTP_HOST"), ""),
		SMTPPort:                  parseInt(state.getenv("SMTP_PORT"), 587),
		SMTPUsername:              defaultString(state.getenv("SMTP_USERNAME"), ""),
		SMTPPassword:              defaultString(state.getenv("SMTP_PASSWORD"), ""),
		SMTPTLS:                   parseBool(state.getenv("SMTP_TLS"), false),
		StaffEmails:               parseStringList(state.getenv("STAFF_EMAILS")),
		AdminSessionTTL:           parseDuration(state.getenv("ADMIN_SESSION_TTL"), 8*time.Hour),
	}

	return cfg, nil
}

// IsMissing returns true if err indicates that one or more required
// configuration values are missing.
func IsMissing(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrMissingConfig)
}

// ErrMissingConfig is returned by Load when required configuration is absent.
var ErrMissingConfig = errors.New("missing required configuration")

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseStringList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseBool(s string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseInt32(s string, fallback int32) int32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(v)
}

func parseInt(s string, fallback int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return fallback
	}
	return int(v)
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return v
}
