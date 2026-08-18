package auth

import (
	"context"
	"crypto/sha512"
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
	"golang.org/x/crypto/bcrypt"
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
	if _, err := testDB.Exec(ctx, "TRUNCATE TABLE audit_events, sessions, email_codes, users, invite_codes RESTART IDENTITY CASCADE"); err != nil {
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
		StaffUsernames:  []string{"admin"},
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
	return NewService(newTestConfig(), repo, mailer, limiter, &StubTurnstile{}), mailer
}

// ensureStaff creates the test admin via the repository (bypassing the invite
// requirement) and returns its id. Username "admin" is staff in newTestConfig.
func ensureStaff(t *testing.T, svc *Service) string {
	t.Helper()
	ctx := context.Background()
	if u, err := svc.repo.GetUserByUsername(ctx, "admin"); err != nil {
		t.Fatalf("lookup admin: %v", err)
	} else if u != nil {
		return u.ID
	}
	sum := sha512.Sum384([]byte("password123"))
	hash, err := bcrypt.GenerateFromPassword(sum[:], bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash admin password: %v", err)
	}
	u, err := svc.repo.CreateUser(ctx, "admin", string(hash))
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	return u.ID
}

func freshInviteCode(t *testing.T, svc *Service) string {
	t.Helper()
	staffID := ensureStaff(t, svc)
	inv, err := svc.CreateInviteCode(context.Background(), staffID, nil)
	if err != nil {
		t.Fatalf("create invite code: %v", err)
	}
	return inv.Code
}

func TestRegister_Success(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "Alice_1", "password123", "tok", code, "127.0.0.1", "fp", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token")
	}
	if tokens.RefreshToken == "" {
		t.Error("expected refresh token")
	}

	user, err := svc.repo.GetUserByUsername(ctx, "alice_1")
	if err != nil {
		t.Fatalf("lookup user: %v", err)
	}
	if user == nil {
		t.Fatal("expected user")
	}
	sum := sha512.Sum384([]byte("password123"))
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), sum[:]); err != nil {
		t.Errorf("stored password hash does not match password: %v", err)
	}
}

func TestRegister_UsernameTaken(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "bob", "password123", "tok", code, "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	code2 := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "Bob", "password123", "tok", code2, "127.0.0.1", "fp2", "ua"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestRegister_CaptchaFails(t *testing.T) {
	cleanTables(t)
	cfg := newTestConfig()
	repo := NewPostgresRepository(testDB)
	limiter := NewMemoryLimiter()
	svc := NewService(cfg, repo, nil, limiter, &StubTurnstile{Fail: true})
	ctx := context.Background()

	if _, err := svc.Register(ctx, "carol", "password123", "tok", "", "127.0.0.1", "fp", "ua"); !errors.Is(err, ErrCaptchaFailed) {
		t.Fatalf("expected ErrCaptchaFailed, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "dave", "password123", "tok", code, "127.0.0.1", "fp", "ua"); err != nil {
		t.Fatalf("register: %v", err)
	}
	tokens, err := svc.Login(ctx, "dave", "password123", "tok", "127.0.0.1", "fp", "ua")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "erin", "password123", "tok", code, "127.0.0.1", "fp", "ua"); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := svc.Login(ctx, "erin", "wrongpass", "tok", "127.0.0.1", "fp", "ua"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Login(ctx, "nobody", "password123", "tok", "127.0.0.1", "fp", "ua"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyPasswordForDeletion(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "frank", "password123", "tok", code, "127.0.0.1", "fp", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.VerifyPasswordForDeletion(ctx, tokens.UserID, "password123"); err != nil {
		t.Fatalf("expected nil for correct password, got %v", err)
	}
	if err := svc.VerifyPasswordForDeletion(ctx, tokens.UserID, "wrongpass"); !errors.Is(err, ErrDeletionInvalidPassword) {
		t.Fatalf("expected ErrDeletionInvalidPassword, got %v", err)
	}
}

func TestRefreshSession_RotatesToken(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "rotate", "password123", "tok", code, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
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
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "revoked", "password123", "tok", code, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
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
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "delete", "password123", "tok", code, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
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
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "delete2", "password123", "tok", code, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
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
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(ctx, "purge", "password123", "tok", code, "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
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

func TestRegister_InviteCode_Invalid(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "noinvite", "password123", "tok", "NOPE-NOPE", "127.0.0.1", "fp", "ua"); !errors.Is(err, ErrInviteCodeInvalid) {
		t.Fatalf("expected ErrInviteCodeInvalid, got %v", err)
	}
}

func TestRegister_InviteCode_Used(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	code := freshInviteCode(t, svc)

	if _, err := svc.Register(ctx, "usercode", "password123", "tok", code, "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := svc.Register(ctx, "usercode2", "password123", "tok", code, "127.0.0.1", "fp2", "ua"); !errors.Is(err, ErrInviteCodeUsed) {
		t.Fatalf("expected ErrInviteCodeUsed, got %v", err)
	}
}

func TestRegister_InviteCode_Expired(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	staffID := ensureStaff(t, svc)
	past := time.Now().UTC().Add(-time.Hour)
	inv, err := svc.CreateInviteCode(ctx, staffID, &past)
	if err != nil {
		t.Fatalf("create expired invite: %v", err)
	}

	if _, err := svc.Register(ctx, "expireduser", "password123", "tok", inv.Code, "127.0.0.1", "fp", "ua"); !errors.Is(err, ErrInviteCodeExpired) {
		t.Fatalf("expected ErrInviteCodeExpired, got %v", err)
	}
}

func TestRegister_UsernameTaken_DoesNotConsumeInviteCode(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	code := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "alice", "password123", "tok", code, "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("first register: %v", err)
	}

	second := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "alice", "password123", "tok", second, "127.0.0.1", "fp2", "ua"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}

	if _, err := svc.Register(ctx, "bob", "password123", "tok", second, "127.0.0.1", "fp3", "ua"); err != nil {
		t.Fatalf("expected code %s to remain usable, register failed: %v", second, err)
	}
}

func TestRegister_InviteCode_ConcurrentSingleUse(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	code := freshInviteCode(t, svc)

	start := make(chan struct{})
	results := make(chan error, 2)
	for _, name := range []string{"winner", "loser"} {
		go func(name string) {
			<-start
			_, err := svc.Register(ctx, name, "password123", "tok", code, "127.0.0.1", "fp-"+name, "ua")
			results <- err
		}(name)
	}
	close(start)

	var ok, used int
	for i := 0; i < 2; i++ {
		err := <-results
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrInviteCodeUsed):
			used++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || used != 1 {
		t.Fatalf("expected 1 success and 1 ErrInviteCodeUsed, got ok=%d used=%d", ok, used)
	}

	inv, err := svc.repo.GetInviteCode(ctx, code)
	if err != nil {
		t.Fatalf("lookup consumed code: %v", err)
	}
	if inv.UsedBy == nil {
		t.Error("expected invite code to be marked used")
	}
}

func TestRegister_InviteCode_NormalizesCase(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	code := freshInviteCode(t, svc)

	if _, err := svc.Register(ctx, "loweruser", "password123", "tok", strings.ToLower(code), "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("register with lowercase invite: %v", err)
	}
	if _, err := svc.Register(ctx, "loweruser2", "password123", "tok", strings.ToLower(code), "127.0.0.1", "fp2", "ua"); !errors.Is(err, ErrInviteCodeUsed) {
		t.Fatalf("expected ErrInviteCodeUsed on reused lowercase code, got %v", err)
	}
}

func TestCreateInviteCode_GeneratesCode(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	staffID := ensureStaff(t, svc)

	inv, err := svc.CreateInviteCode(ctx, staffID, nil)
	if err != nil {
		t.Fatalf("create invite code: %v", err)
	}
	if inv.ID == "" {
		t.Error("expected id")
	}
	if len(inv.Code) < 8 {
		t.Errorf("expected non-trivial code, got %q", inv.Code)
	}
	if inv.UsedAt != nil {
		t.Error("expected unused code")
	}
	got, err := svc.repo.GetInviteCode(ctx, inv.Code)
	if err != nil {
		t.Fatalf("lookup invite code: %v", err)
	}
	if got == nil || got.ID != inv.ID {
		t.Error("expected stored invite code to be retrievable")
	}
}

func TestRevokeInviteCode_UnusedDeletesUsedRetries(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	staffID := ensureStaff(t, svc)

	inv, err := svc.CreateInviteCode(ctx, staffID, nil)
	if err != nil {
		t.Fatalf("create invite code: %v", err)
	}
	if err := svc.RevokeInviteCode(ctx, inv.ID, staffID); err != nil {
		t.Fatalf("revoke unused: %v", err)
	}
	if err := svc.RevokeInviteCode(ctx, inv.ID, staffID); !errors.Is(err, ErrInviteCodeNotFound) {
		t.Fatalf("expected ErrInviteCodeNotFound after revoke, got %v", err)
	}

	code := freshInviteCode(t, svc)
	if _, err := svc.Register(ctx, "consumed", "password123", "tok", code, "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("register: %v", err)
	}
	used, err := svc.repo.GetInviteCode(ctx, code)
	if err != nil {
		t.Fatalf("lookup used code: %v", err)
	}
	if err := svc.RevokeInviteCode(ctx, used.ID, staffID); !errors.Is(err, ErrInviteCodeUsed) {
		t.Fatalf("expected ErrInviteCodeUsed on used code, got %v", err)
	}
}

func TestListInviteCodes_Pagination(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	staffID := ensureStaff(t, svc)

	created := make([]*InviteCode, 3)
	base := time.Now().UTC().Add(-3 * time.Minute)
	for i := 0; i < 3; i++ {
		inv, err := svc.CreateInviteCode(ctx, staffID, nil)
		if err != nil {
			t.Fatalf("create invite code %d: %v", i, err)
		}
		created[i] = inv
	}
	// Backdate each code to a distinct created_at so the strict
	// created_at < cursor predicate cannot skip a row on ties.
	for i, inv := range created {
		ts := base.Add(time.Duration(i) * time.Second)
		if _, err := testDB.Exec(ctx, "UPDATE invite_codes SET created_at = $2 WHERE id = $1", inv.ID, ts); err != nil {
			t.Fatalf("backdate invite %s: %v", inv.ID, err)
		}
	}

	codes, nextCursor, hasMore, err := svc.ListInviteCodes(ctx, "", 2)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}
	if !hasMore || nextCursor == nil {
		t.Error("expected hasMore and a next cursor")
	}
	if !codes[0].CreatedAt.After(codes[1].CreatedAt) {
		t.Error("expected page 1 codes ordered newest-first")
	}

	codes2, nextCursor2, hasMore2, err := svc.ListInviteCodes(ctx, *nextCursor, 2)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(codes2) != 1 {
		t.Fatalf("expected 1 code, got %d", len(codes2))
	}
	if hasMore2 {
		t.Error("expected no more pages")
	}
	if nextCursor2 != nil {
		t.Errorf("expected nil next cursor, got %q", *nextCursor2)
	}
	if !codes2[0].CreatedAt.Before(codes[0].CreatedAt) {
		t.Error("expected page 2 code strictly older than page 1 (disjoint pages)")
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
