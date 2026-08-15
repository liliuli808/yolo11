package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yiguan/api/internal/auth"
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
		fmt.Println("TEST_DATABASE_URL or DATABASE_URL required for identity tests")
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

	if err := resetIdentitySchema(ctx, pool); err != nil {
		fmt.Printf("reset identity schema: %v\n", err)
		os.Exit(1)
	}
	if err := runMigrations(ctx, pool); err != nil {
		fmt.Printf("run migrations: %v\n", err)
		os.Exit(1)
	}

	testDB = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func resetIdentitySchema(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`DROP TABLE IF EXISTS idempotency_keys`,
		`DROP TABLE IF EXISTS saves`,
		`DROP TABLE IF EXISTS media_assets`,
		`DROP TABLE IF EXISTS reactions`,
		`DROP TABLE IF EXISTS comments`,
		`DROP TABLE IF EXISTS posts`,
		`DROP TABLE IF EXISTS topic_follows`,
		`DROP TABLE IF EXISTS topics`,
		`DROP TABLE IF EXISTS data_exports`,
		`DROP TABLE IF EXISTS moderation_actions`,
		`DROP TABLE IF EXISTS case_reports`,
		`DROP TABLE IF EXISTS moderation_cases`,
		`DROP TABLE IF EXISTS reports`,
		`DROP TABLE IF EXISTS blocks`,
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_default_persona`,
		`DROP TABLE IF EXISTS personas`,
		`ALTER TABLE users DROP COLUMN IF EXISTS default_persona_id`,
		`ALTER TABLE users DROP COLUMN IF EXISTS max_personas`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("reset schema: %s: %w", stmt, err)
		}
	}
	return nil
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	for _, dir := range []string{
		filepath.Join("..", "..", "migrations", "auth"),
		filepath.Join("..", "..", "migrations", "identity"),
		filepath.Join("..", "..", "migrations", "content"),
		filepath.Join("..", "..", "migrations", "moderation"),
	} {
		files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
		if err != nil {
			return fmt.Errorf("glob migrations %s: %w", dir, err)
		}
		sort.Strings(files)
		for _, rel := range files {
			data, err := os.ReadFile(rel)
			if err != nil {
				return fmt.Errorf("read migration %s: %w", rel, err)
			}
			for _, stmt := range strings.Split(string(data), ";") {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if _, err := pool.Exec(ctx, stmt); err != nil {
					return fmt.Errorf("execute migration %s: %w", rel, err)
				}
			}
		}
	}
	return nil
}

func cleanTables(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := testDB.Exec(ctx, "TRUNCATE TABLE moderation_actions, case_reports, moderation_cases, reports, blocks, data_exports, personas, audit_events, sessions, email_codes, users RESTART IDENTITY CASCADE"); err != nil {
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
	lastTo   string
	lastCode string
}

func (m *recordingMailer) SendEmailCode(ctx context.Context, to, code string, expiresIn time.Duration) error {
	m.lastTo = to
	m.lastCode = code
	return nil
}

func (m *recordingMailer) LastCode() string { return m.lastCode }

func newTestService() (*Service, *recordingMailer) {
	mailer := &recordingMailer{}
	authRepo := auth.NewPostgresRepository(testDB)
	idRepo := NewPostgresRepository(testDB)
	return NewService(newTestConfig(), idRepo, authRepo, mailer, auth.NewMemoryLimiter(), nil), mailer
}

func createUser(t *testing.T, email string) *auth.User {
	t.Helper()
	ctx := context.Background()
	u, err := auth.NewPostgresRepository(testDB).FindOrCreateUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func TestService_GetMe(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	u := createUser(t, "me@example.com")

	profile, err := svc.GetMe(ctx, u.ID)
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if profile.ID != u.ID {
		t.Errorf("expected id %s, got %s", u.ID, profile.ID)
	}
	if profile.EmailNormalized != "me@example.com" {
		t.Errorf("expected email me@example.com, got %s", profile.EmailNormalized)
	}
	if profile.Status != "active" {
		t.Errorf("expected active, got %s", profile.Status)
	}
	if profile.PersonaCount != 0 {
		t.Errorf("expected 0 personas, got %d", profile.PersonaCount)
	}
	if profile.MaxPersonas != 5 {
		t.Errorf("expected max 5, got %d", profile.MaxPersonas)
	}
	if profile.DefaultPersonaID != nil {
		t.Error("expected no default persona")
	}
	if profile.DeletionGracePeriodEndsAt != nil {
		t.Error("expected no deletion state")
	}
}

func TestService_GetMe_NotFound(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()

	_, err := svc.GetMe(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestService_CreatePersona(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "persona@example.com")

	bio := "hello"
	p, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{
		Alias:       "alice",
		Bio:         &bio,
		AvatarSeed:  "seed1",
		AvatarColor: "#FF5733",
	})
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	if p.Alias != "alice" {
		t.Errorf("expected alias alice, got %s", p.Alias)
	}
	if p.Status != "active" {
		t.Errorf("expected active, got %s", p.Status)
	}
	if !p.IsDefault {
		t.Error("expected first persona to be default")
	}

	profile, err := svc.GetMe(ctx, u.ID)
	if err != nil {
		t.Fatalf("get me after create: %v", err)
	}
	if profile.PersonaCount != 1 {
		t.Errorf("expected persona count 1, got %d", profile.PersonaCount)
	}
	if profile.DefaultPersonaID == nil || *profile.DefaultPersonaID != p.ID {
		t.Error("expected default persona id set")
	}
}

func TestService_CreatePersona_AliasTaken(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "alias@example.com")

	if _, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "taken"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "taken"})
	if !errors.Is(err, ErrPersonaAliasTaken) {
		t.Fatalf("expected ErrPersonaAliasTaken, got %v", err)
	}
}

func TestService_CreatePersona_MaxReached(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "max@example.com")

	for i := 0; i < 5; i++ {
		if _, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: fmt.Sprintf("p%d", i)}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	_, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "overflow"})
	if !errors.Is(err, ErrPersonaMaxReached) {
		t.Fatalf("expected ErrPersonaMaxReached, got %v", err)
	}
}

func TestService_ArchivePersona(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "archive@example.com")

	p, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "arch"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ArchivePersona(ctx, u.ID, p.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	pAgain, err := svc.GetPrivatePersona(ctx, u.ID, p.ID)
	if err != nil {
		t.Fatalf("get archived persona: %v", err)
	}
	if pAgain.Status != "archived" {
		t.Errorf("expected archived status, got %s", pAgain.Status)
	}

	profile, err := svc.GetMe(ctx, u.ID)
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if profile.PersonaCount != 0 {
		t.Errorf("expected count 0 after archive, got %d", profile.PersonaCount)
	}
	if profile.DefaultPersonaID != nil {
		t.Error("expected default persona cleared when no active personas remain")
	}
}

func TestService_SetDefaultPersona(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "default@example.com")

	first, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "first"})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "second"})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if !first.IsDefault {
		t.Error("expected first to be default initially")
	}

	updated, err := svc.SetDefaultPersona(ctx, u.ID, second.ID)
	if err != nil {
		t.Fatalf("set default: %v", err)
	}
	if !updated.IsDefault {
		t.Error("expected second to be default")
	}

	firstAgain, err := svc.GetPrivatePersona(ctx, u.ID, first.ID)
	if err != nil {
		t.Fatalf("get first: %v", err)
	}
	if firstAgain.IsDefault {
		t.Error("expected first to no longer be default")
	}
}

func TestService_SetDefaultPersona_Restricted(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "restricted-default@example.com")

	p, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "restricted"})
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	if _, err := testDB.Exec(ctx, "UPDATE personas SET status = 'restricted' WHERE id = $1", p.ID); err != nil {
		t.Fatalf("set restricted status: %v", err)
	}

	if _, err := svc.SetDefaultPersona(ctx, u.ID, p.ID); !errors.Is(err, ErrPersonaRestricted) {
		t.Fatalf("expected ErrPersonaRestricted, got %v", err)
	}
}

func TestService_GetPublicPersona_HidesRealProfile(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "public@example.com")

	p, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "public"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	public, err := svc.GetPublicPersona(ctx, p.ID)
	if err != nil {
		t.Fatalf("get public persona: %v", err)
	}
	if public.ID != p.ID {
		t.Errorf("expected id %s, got %s", p.ID, public.ID)
	}
	if public.Alias != "public" {
		t.Errorf("expected alias public, got %s", public.Alias)
	}
	// Ensure no real-profile fields leak in the JSON representation.
	data, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("marshal public persona: %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("unmarshal public persona: %v", err)
	}
	if _, ok := envelope["realProfileId"]; ok {
		t.Error("public persona must not expose real profile id")
	}
	if _, ok := envelope["real_profile_id"]; ok {
		t.Error("public persona must not expose real profile id")
	}
}

func TestService_GetPublicPersona_ArchivedNotFound(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "archivepub@example.com")

	p, err := svc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "archpub"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.ArchivePersona(ctx, u.ID, p.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	_, err = svc.GetPublicPersona(ctx, p.ID)
	if !errors.Is(err, ErrPersonaNotFound) {
		t.Fatalf("expected not found for archived persona, got %v", err)
	}
}

func TestService_EmailChange(t *testing.T) {
	cleanTables(t)
	svc, mailer := newTestService()
	ctx := context.Background()
	u := createUser(t, "old@example.com")

	if err := svc.RequestEmailChange(ctx, u.ID, "new@example.com", "127.0.0.1", "fp1"); err != nil {
		t.Fatalf("request email change: %v", err)
	}
	if mailer.LastCode() == "" {
		t.Fatal("expected verification code to be sent")
	}

	profile, err := svc.ConfirmEmailChange(ctx, u.ID, "new@example.com", mailer.LastCode())
	if err != nil {
		t.Fatalf("confirm email change: %v", err)
	}
	if profile.EmailNormalized != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", profile.EmailNormalized)
	}

	authRepo := auth.NewPostgresRepository(testDB)
	updated, err := authRepo.GetUserByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("lookup user: %v", err)
	}
	if updated.EmailNormalized != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", updated.EmailNormalized)
	}
}

func TestService_EmailChange_EmailAlreadyUsed(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	createUser(t, "existing@example.com")
	u := createUser(t, "change@example.com")

	err := svc.RequestEmailChange(ctx, u.ID, "existing@example.com", "127.0.0.1", "fp1")
	if !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Fatalf("expected ErrEmailAlreadyUsed, got %v", err)
	}
}

func TestService_EmailChange_RateLimited(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "email-change-rate@example.com")

	for i := 0; i < 5; i++ {
		if err := svc.RequestEmailChange(ctx, u.ID, "target@example.com", "127.0.0.1", "fp1"); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}

	err := svc.RequestEmailChange(ctx, u.ID, "target@example.com", "127.0.0.1", "fp1")
	var rateLimitErr *auth.RateLimitError
	if !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if rateLimitErr.RetryAfter <= 0 {
		t.Error("expected positive retry after")
	}
}

func TestService_DataExport(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "export@example.com")

	exp, err := svc.RequestDataExport(ctx, u.ID, "json")
	if err != nil {
		t.Fatalf("request export: %v", err)
	}
	if exp.Status != "pending" {
		t.Errorf("expected pending, got %s", exp.Status)
	}

	got, err := svc.GetDataExport(ctx, u.ID, exp.ID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if got.ID != exp.ID {
		t.Errorf("expected id %s, got %s", exp.ID, got.ID)
	}
}

func TestService_DataExport_InvalidFormat(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "export-bad-format@example.com")

	_, err := svc.RequestDataExport(ctx, u.ID, "xml")
	if !errors.Is(err, ErrInvalidExportFormat) {
		t.Fatalf("expected ErrInvalidExportFormat, got %v", err)
	}
}

func TestService_DataExport_NotFound(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "export-notfound@example.com")

	_, err := svc.GetDataExport(ctx, u.ID, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrExportNotFound) {
		t.Fatalf("expected ErrExportNotFound, got %v", err)
	}
}

func TestService_DataExport_RateLimited(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "exportrate@example.com")

	if _, err := svc.RequestDataExport(ctx, u.ID, "json"); err != nil {
		t.Fatalf("first export: %v", err)
	}
	_, err := svc.RequestDataExport(ctx, u.ID, "json")
	if !errors.Is(err, ErrExportRateLimited) {
		t.Fatalf("expected ErrExportRateLimited, got %v", err)
	}
}

func TestService_ProcessPendingExports(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	ctx := context.Background()
	u := createUser(t, "export-pending@example.com")

	exp, err := svc.RequestDataExport(ctx, u.ID, "json")
	if err != nil {
		t.Fatalf("request export: %v", err)
	}

	// Force the export to be older than the grace period.
	if _, err := testDB.Exec(ctx, "UPDATE data_exports SET requested_at = now() - interval '10 minutes' WHERE id = $1", exp.ID); err != nil {
		t.Fatalf("age export: %v", err)
	}

	processed, err := svc.ProcessPendingExports(ctx)
	if err != nil {
		t.Fatalf("process pending exports: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 export processed, got %d", processed)
	}

	ready, err := svc.GetDataExport(ctx, u.ID, exp.ID)
	if err != nil {
		t.Fatalf("get export: %v", err)
	}
	if ready.Status != "ready" {
		t.Errorf("expected ready, got %s", ready.Status)
	}
	if ready.DownloadURL == nil || *ready.DownloadURL == "" {
		t.Error("expected download url")
	}
	if ready.ExpiresAt == nil {
		t.Error("expected expires at")
	}
}

func TestService_AccountDeletionHook_ArchivesPersonas(t *testing.T) {
	cleanTables(t)
	cfg := newTestConfig()
	authRepo := auth.NewPostgresRepository(testDB)
	idRepo := NewPostgresRepository(testDB)
	mailer := &recordingMailer{}
	authLimiter := auth.NewMemoryLimiter()
	authSvc := auth.NewService(cfg, authRepo, mailer, authLimiter)
	idSvc := NewService(cfg, idRepo, authRepo, mailer, auth.NewMemoryLimiter(), nil)
	authSvc.IdentityCleanup = idSvc.CleanupOnAccountDeletion
	ctx := context.Background()

	u, err := authRepo.FindOrCreateUserByEmail(ctx, "hook@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	p, err := idSvc.CreatePersona(ctx, u.ID, &PersonaCreateRequest{Alias: "hookpersona"})
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}

	if _, err := authSvc.RequestAccountDeletion(ctx, u.ID); err != nil {
		t.Fatalf("request account deletion: %v", err)
	}

	_, err = idSvc.GetPublicPersona(ctx, p.ID)
	if !errors.Is(err, ErrPersonaNotFound) {
		t.Fatalf("expected persona archived after deletion, got %v", err)
	}
}
