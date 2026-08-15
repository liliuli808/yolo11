package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yiguan/api/internal/platform/config"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		fmt.Println("TEST_DATABASE_URL or DATABASE_URL required for auth tests")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Printf("connect test database: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("ping test database: %v\n", err)
		os.Exit(1)
	}

	if err := runMigrations(ctx, pool); err != nil {
		fmt.Printf("run auth migrations: %v\n", err)
		os.Exit(1)
	}

	testDB = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "auth", "*.up.sql"))
	if err != nil {
		return fmt.Errorf("glob auth migrations: %w", err)
	}
	sort.Strings(files)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration file: %w", err)
		}

		for _, stmt := range strings.Split(string(data), ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("execute migration statement: %w", err)
			}
		}
	}
	return nil
}

func cleanTables(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := testDB.Exec(ctx, "TRUNCATE TABLE audit_events, sessions, email_codes, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("clean tables: %v", err)
	}
}

func newTestConfig() *config.Config {
	return &config.Config{
		JWTSigningKey:   "jwt-secret-key-for-tests-only",
		EmailCodeKey:    "email-code-secret-key-for-tests-only",
		EmailFrom:       "noreply@example.com",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
}

type recordingMailer struct {
	mu       sync.Mutex
	codes    []string
	lastTo   string
	lastCode string
}

func (m *recordingMailer) SendEmailCode(ctx context.Context, to, code string, expiresIn time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes = append(m.codes, code)
	m.lastTo = to
	m.lastCode = code
	return nil
}

func (m *recordingMailer) LastCode() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastCode
}

func newTestService() (*Service, *recordingMailer) {
	mailer := &recordingMailer{}
	repo := NewPostgresRepository(testDB)
	limiter := NewMemoryLimiter()
	return NewService(newTestConfig(), repo, mailer, limiter), mailer
}

func TestCreateEmailSession_Success(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "alice@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}

	code := mailer.LastCode()
	if code == "" {
		t.Fatal("no code sent")
	}

	tokens, err := svc.CreateEmailSession(ctx, email, code, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token")
	}
	if tokens.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if tokens.TokenType != "Bearer" {
		t.Errorf("expected Bearer token type, got %q", tokens.TokenType)
	}
}

func TestCreateEmailSession_CodeReuseFails(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "reuse@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	code := mailer.LastCode()

	if _, err := svc.CreateEmailSession(ctx, email, code, "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("first create session: %v", err)
	}

	if _, err := svc.CreateEmailSession(ctx, email, code, "127.0.0.1", "fp1", "ua"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode on reuse, got %v", err)
	}
}

func TestCreateEmailSession_ExpiredCodeFails(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "expired@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	code := mailer.LastCode()

	// Force the latest code to expire by updating the database directly.
	if _, err := testDB.Exec(ctx, "UPDATE email_codes SET expires_at = now() - interval '1 minute' WHERE email = $1", NormalizeEmail(email)); err != nil {
		t.Fatalf("expire code: %v", err)
	}

	if _, err := svc.CreateEmailSession(ctx, email, code, "127.0.0.1", "fp1", "ua"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expected ErrInvalidCode for expired code, got %v", err)
	}
}

func TestRequestEmailCode_RateLimitedByEmail(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	email := "rate@example.com"

	for i := 0; i < 5; i++ {
		if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}

	err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1")
	var rateLimitErr *RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if rateLimitErr.RetryAfter <= 0 {
		t.Error("expected positive retry after")
	}
}

func TestRefreshSession_RotatesToken(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "rotate@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	tokens, err := svc.CreateEmailSession(ctx, email, mailer.LastCode(), "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	result, err := svc.RefreshSession(ctx, tokens.RefreshToken, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("refresh session: %v", err)
	}
	if result.RefreshToken == tokens.RefreshToken {
		t.Error("expected a new refresh token after rotation")
	}
	if result.AccessToken == "" {
		t.Error("expected new access token")
	}

	// Old refresh token must no longer be accepted.
	if _, err := svc.RefreshSession(ctx, tokens.RefreshToken, "127.0.0.1", "fp1", "ua"); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired for old refresh token, got %v", err)
	}
}

func TestRefreshSession_RevokedSessionRejected(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "revoked@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	tokens, err := svc.CreateEmailSession(ctx, email, mailer.LastCode(), "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Revoke the session using its database id. We need to look it up from the refresh hash.
	oldHash := hashRefreshToken(tokens.RefreshToken)
	row := testDB.QueryRow(ctx, "SELECT id FROM sessions WHERE refresh_token_hash = $1", oldHash)
	var sessionID string
	if err := row.Scan(&sessionID); err != nil {
		t.Fatalf("lookup session: %v", err)
	}
	if err := svc.Logout(ctx, sessionID); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := svc.RefreshSession(ctx, tokens.RefreshToken, "127.0.0.1", "fp1", "ua"); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired after revocation, got %v", err)
	}
}

func TestRequestAccountDeletion_RevokesAllSessions(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "delete@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	tokens, err := svc.CreateEmailSession(ctx, email, mailer.LastCode(), "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	user, err := svc.repo.GetUserByID(ctx, UserIDFromToken(ctx, svc, tokens.AccessToken))
	if err != nil {
		t.Fatalf("lookup user: %v", err)
	}

	state, err := svc.RequestAccountDeletion(ctx, user.ID)
	if err != nil {
		t.Fatalf("request deletion: %v", err)
	}
	if state.GracePeriodEndsAt.IsZero() {
		t.Error("expected grace period end")
	}

	if _, err := svc.RefreshSession(ctx, tokens.RefreshToken, "127.0.0.1", "fp1", "ua"); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired after account deletion, got %v", err)
	}
}

func TestRequestAccountDeletion_AlreadyPending(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "delete2@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	tokens, err := svc.CreateEmailSession(ctx, email, mailer.LastCode(), "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	userID := UserIDFromToken(ctx, svc, tokens.AccessToken)
	if _, err := svc.RequestAccountDeletion(ctx, userID); err != nil {
		t.Fatalf("first deletion request: %v", err)
	}
	if _, err := svc.RequestAccountDeletion(ctx, userID); !errors.Is(err, ErrDeletionAlreadyPending) {
		t.Fatalf("expected ErrDeletionAlreadyPending, got %v", err)
	}
}

func TestPurgeDeletedAccounts_PurgesPastGracePeriod(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()

	email := "purge@example.com"
	if err := svc.RequestEmailCode(ctx, email, "login", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request code: %v", err)
	}
	tokens, err := svc.CreateEmailSession(ctx, email, mailer.LastCode(), "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	userID := UserIDFromToken(ctx, svc, tokens.AccessToken)

	if _, err := svc.RequestAccountDeletion(ctx, userID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}

	// Move the grace period into the past.
	if _, err := testDB.Exec(ctx, "UPDATE users SET deletion_grace_period_ends_at = now() - interval '1 minute' WHERE id = $1", userID); err != nil {
		t.Fatalf("expire grace period: %v", err)
	}

	purged, err := svc.PurgeDeletedAccounts(ctx)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if purged != 1 {
		t.Fatalf("expected 1 purged, got %d", purged)
	}

	user, err := svc.repo.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatalf("lookup user: %v", err)
	}
	if user.Status != "deleted" {
		t.Errorf("expected deleted, got %s", user.Status)
	}
}

// UserIDFromToken is a test helper that extracts the user id from an access token.
func UserIDFromToken(ctx context.Context, svc *Service, token string) string {
	claims, err := svc.ValidateAccessToken(ctx, token)
	if err != nil {
		panic(err)
	}
	return claims.UserID
}
