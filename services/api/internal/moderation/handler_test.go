package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/content"
	"github.com/yiguan/api/internal/identity"
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
		fmt.Println("TEST_DATABASE_URL or DATABASE_URL required for moderation tests")
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

	if err := resetModerationSchema(ctx, pool); err != nil {
		fmt.Printf("reset moderation schema: %v\n", err)
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

func resetModerationSchema(ctx context.Context, pool *pgxpool.Pool) error {
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
	if _, err := testDB.Exec(ctx, "TRUNCATE TABLE moderation_actions, case_reports, moderation_cases, reports, blocks, idempotency_keys, saves, media_assets, reactions, comments, posts, topic_follows, topics, data_exports, personas, audit_events, sessions, email_codes, users RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("clean tables: %v", err)
	}
	reseedBaseTopics(t)
}

func reseedBaseTopics(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	const sql = `
		INSERT INTO topics (name, description, category, status, slug)
		VALUES
			('General', 'Open conversation about anything.', 'Everyday', 'active', 'general'),
			('Reflection', 'Deeper thoughts and personal reflections.', 'Reflection', 'active', 'reflection'),
			('Creative', 'Share creative work and inspiration.', 'Creative', 'active', 'creative')
		ON CONFLICT (name) DO NOTHING
	`
	if _, err := testDB.Exec(ctx, sql); err != nil {
		t.Fatalf("reseed topics: %v", err)
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
	lastTo   string
	lastCode string
}

func (m *recordingMailer) SendEmailCode(ctx context.Context, to, code string, expiresIn time.Duration) error {
	m.lastTo = to
	m.lastCode = code
	return nil
}

func (m *recordingMailer) LastCode() string { return m.lastCode }

type testFixtures struct {
	authHandler       *auth.Handler
	idHandler         *identity.Handler
	contentHandler    *content.Handler
	moderationHandler *Handler
}

func setupHandlerTest(t *testing.T) *testFixtures {
	cleanTables(t)
	cfg := newTestConfig()
	authRepo := auth.NewPostgresRepository(testDB)
	idRepo := identity.NewPostgresRepository(testDB)
	mailer := &recordingMailer{}
	authLimiter := auth.NewMemoryLimiter()
	authSvc := auth.NewService(cfg, authRepo, mailer, authLimiter, &auth.StubTurnstile{})
	idLimiter := auth.NewMemoryLimiter()
	idSvc := identity.NewService(cfg, idRepo, authRepo, mailer, idLimiter, nil)
	authHandler := auth.NewHandler(authSvc, cfg)
	idHandler := identity.NewHandler(idSvc, authHandler, cfg)
	contentRepo := content.NewPostgresRepository(testDB)
	contentLimiter := auth.NewMemoryLimiter()
	moderationRepo := NewPostgresRepository(testDB)
	moderationLimiter := auth.NewMemoryLimiter()
	moderationSvc := NewService(cfg, moderationRepo, idRepo, moderationLimiter)
	contentSvc := content.NewService(cfg, contentRepo, idRepo, moderationSvc, contentLimiter)
	contentHandler := content.NewHandler(contentSvc, idHandler, idSvc, cfg)
	moderationHandler := NewHandler(moderationSvc, authHandler, idHandler, idSvc, cfg)
	return &testFixtures{authHandler: authHandler, idHandler: idHandler, contentHandler: contentHandler, moderationHandler: moderationHandler}
}

func mountModerationHandler(f *testFixtures) http.Handler {
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		f.authHandler.Mount(v1)
		f.idHandler.Mount(v1)
		f.contentHandler.Mount(v1)
		f.moderationHandler.Mount(v1)
	})
	return r
}

func createSession(t *testing.T, serverURL string, username, password string) string {
	t.Helper()
	inviteCode := freshInviteCode(t)
	body, _ := json.Marshal(map[string]any{"username": username, "password": password, "turnstileToken": "tok", "inviteCode": inviteCode})
	req, _ := http.NewRequest(http.MethodPost, serverURL+"/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "register-"+username)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var session map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	token, _ := session["accessToken"].(string)
	if token == "" {
		t.Fatal("missing access token")
	}
	return token
}

func freshInviteCode(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	repo := auth.NewPostgresRepository(testDB)
	issuer, err := repo.GetUserByUsername(ctx, "inviteissuer")
	if err != nil {
		t.Fatalf("lookup invite issuer: %v", err)
	}
	if issuer == nil {
		issuer, err = repo.CreateUser(ctx, "inviteissuer", "unused-test-hash")
		if err != nil {
			t.Fatalf("create invite issuer: %v", err)
		}
	}
	svc := auth.NewService(newTestConfig(), repo, &recordingMailer{}, auth.NewMemoryLimiter(), &auth.StubTurnstile{})
	invite, err := svc.CreateInviteCode(ctx, issuer.ID, nil)
	if err != nil {
		t.Fatalf("create invite code: %v", err)
	}
	return invite.Code
}

func createPersona(t *testing.T, serverURL, token, alias string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"alias": alias})
	req, _ := http.NewRequest(http.MethodPost, serverURL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-persona-key-"+alias)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected created, got %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode persona: %v", err)
	}
	return created["id"].(string)
}

func publishPost(t *testing.T, postID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := testDB.Exec(ctx, "UPDATE posts SET moderation_state = 'published' WHERE id = $1", postID); err != nil {
		t.Fatalf("publish post: %v", err)
	}
}

func getTopicID(t *testing.T, serverURL, name string) string {
	t.Helper()
	resp, err := http.Get(serverURL + "/v1/topics?q=" + name)
	if err != nil {
		t.Fatalf("list topics: %v", err)
	}
	defer resp.Body.Close()
	var page map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode topics: %v", err)
	}
	data := page["data"].([]any)
	for _, item := range data {
		topic := item.(map[string]any)
		if topic["name"] == name {
			return topic["id"].(string)
		}
	}
	t.Fatalf("topic %s not found", name)
	return ""
}

func createPublishedPost(t *testing.T, serverURL, token, personaID, topicID, content string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": content})
	req, _ := http.NewRequest(http.MethodPost, serverURL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "mod-post-create-"+content)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected created, got %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	publishPost(t, created["id"].(string))
	return created["id"].(string)
}

func TestHandler_CreateAndListBlocks(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountModerationHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "blockuser", "password123")
	viewerPersonaID := createPersona(t, server.URL, token, "viewer")
	targetToken := createSession(t, server.URL, "blocktarget", "password123")
	targetPersonaID := createPersona(t, server.URL, targetToken, "target")

	client := &http.Client{Timeout: 5 * time.Second}
	body, _ := json.Marshal(map[string]any{"personaId": targetPersonaID})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/blocks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "block-create-key-1")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create block: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var block map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&block); err != nil {
		t.Fatalf("decode block: %v", err)
	}
	if block["id"] == "" {
		t.Error("expected block id")
	}
	persona := block["persona"].(map[string]any)
	if persona["id"] != targetPersonaID {
		t.Errorf("expected blocked persona id %s, got %v", targetPersonaID, persona["id"])
	}

	_ = viewerPersonaID

	// List blocks.
	listReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/me/blocks", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listResp, err := client.Do(listReq)
	if err != nil {
		t.Fatalf("list blocks: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, listResp.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("decode blocks page: %v", err)
	}
	data := page["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 block, got %d", len(data))
	}
}

func TestHandler_BlockSelfIsRejected(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountModerationHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "blockself", "password123")
	personaID := createPersona(t, server.URL, token, "selfblocker")

	body, _ := json.Marshal(map[string]any{"personaId": personaID})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/blocks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "block-self-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create block: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected %d, got %d", http.StatusUnprocessableEntity, resp.StatusCode)
	}
}

func TestHandler_BlockHidesContentInFeed(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountModerationHandler(f))
	defer server.Close()

	posterToken := createSession(t, server.URL, "blockposter", "password123")
	posterPersonaID := createPersona(t, server.URL, posterToken, "blockedposter")
	topicID := getTopicID(t, server.URL, "General")
	createPublishedPost(t, server.URL, posterToken, posterPersonaID, topicID, "blocked post")

	viewerToken := createSession(t, server.URL, "blockviewer", "password123")
	_ = createPersona(t, server.URL, viewerToken, "viewerfeed")

	// Feed should include post before block.
	beforeResp, err := http.Get(server.URL + "/v1/posts")
	if err != nil {
		t.Fatalf("list posts before block: %v", err)
	}
	defer beforeResp.Body.Close()
	var beforePage map[string]any
	if err := json.NewDecoder(beforeResp.Body).Decode(&beforePage); err != nil {
		t.Fatalf("decode posts before: %v", err)
	}
	if len(beforePage["data"].([]any)) != 1 {
		t.Fatalf("expected 1 post before block, got %d", len(beforePage["data"].([]any)))
	}

	// Block poster.
	body, _ := json.Marshal(map[string]any{"personaId": posterPersonaID})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/blocks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "block-feed-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	blockResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create block: %v", err)
	}
	defer blockResp.Body.Close()
	if blockResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, blockResp.StatusCode)
	}

	// Feed should exclude post after block.
	afterReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/posts", nil)
	afterReq.Header.Set("Authorization", "Bearer "+viewerToken)
	afterResp, err := client.Do(afterReq)
	if err != nil {
		t.Fatalf("list posts after block: %v", err)
	}
	defer afterResp.Body.Close()
	var afterPage map[string]any
	if err := json.NewDecoder(afterResp.Body).Decode(&afterPage); err != nil {
		t.Fatalf("decode posts after: %v", err)
	}
	if len(afterPage["data"].([]any)) != 0 {
		t.Fatalf("expected 0 posts after block, got %d", len(afterPage["data"].([]any)))
	}
}

func TestHandler_CreateAndGetReport(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountModerationHandler(f))
	defer server.Close()

	targetToken := createSession(t, server.URL, "reporttarget", "password123")
	targetPersonaID := createPersona(t, server.URL, targetToken, "reporttarget")

	reporterToken := createSession(t, server.URL, "reporter", "password123")
	_ = createPersona(t, server.URL, reporterToken, "reporter")

	body, _ := json.Marshal(map[string]any{
		"targetType": "persona",
		"targetId":   targetPersonaID,
		"category":   "harassment",
		"details":    "being mean",
	})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/reports", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+reporterToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "report-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var report map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report["category"] != "harassment" {
		t.Errorf("expected harassment, got %v", report["category"])
	}
	if report["status"] != "open" {
		t.Errorf("expected open, got %v", report["status"])
	}

	getReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/reports/"+report["id"].(string), nil)
	getReq.Header.Set("Authorization", "Bearer "+reporterToken)
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, getResp.StatusCode)
	}
}

func TestHandler_ModerationCaseLifecycle(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountModerationHandler(f))
	defer server.Close()

	// Create a post to report.
	posterToken := createSession(t, server.URL, "caseposter", "password123")
	posterPersonaID := createPersona(t, server.URL, posterToken, "caseposter")
	topicID := getTopicID(t, server.URL, "General")
	postID := createPublishedPost(t, server.URL, posterToken, posterPersonaID, topicID, "case post")

	// Submit a report.
	reporterToken := createSession(t, server.URL, "casereporter", "password123")
	_ = createPersona(t, server.URL, reporterToken, "casereporter")
	reportBody, _ := json.Marshal(map[string]any{
		"targetType": "post",
		"targetId":   postID,
		"category":   "spam",
	})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/reports", bytes.NewReader(reportBody))
	req.Header.Set("Authorization", "Bearer "+reporterToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "case-report-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	reportResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create report: %v", err)
	}
	defer reportResp.Body.Close()
	if reportResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d from report, got %d", http.StatusCreated, reportResp.StatusCode)
	}
	var report map[string]any
	if err := json.NewDecoder(reportResp.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}

	// Non-staff cannot access moderation endpoints.
	caseListResp, err := client.Get(server.URL + "/v1/moderation/cases")
	if err != nil {
		t.Fatalf("list cases unauth: %v", err)
	}
	defer caseListResp.Body.Close()
	if caseListResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected %d for unauth cases, got %d", http.StatusUnauthorized, caseListResp.StatusCode)
	}

	// Staff login.
	staffToken := createSession(t, server.URL, "admin", "password123")
	caseBody, _ := json.Marshal(map[string]any{
		"targetType": "post",
		"targetId":   postID,
		"reportIds":  []string{report["id"].(string)},
	})
	caseReq, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/moderation/cases", bytes.NewReader(caseBody))
	caseReq.Header.Set("Authorization", "Bearer "+staffToken)
	caseReq.Header.Set("Content-Type", "application/json")
	caseReq.Header.Set("Idempotency-Key", "case-create-key-1")
	caseResp, err := client.Do(caseReq)
	if err != nil {
		t.Fatalf("create case: %v", err)
	}
	defer caseResp.Body.Close()
	if caseResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d from case, got %d", http.StatusCreated, caseResp.StatusCode)
	}
	var createdCase map[string]any
	if err := json.NewDecoder(caseResp.Body).Decode(&createdCase); err != nil {
		t.Fatalf("decode case: %v", err)
	}
	caseID := createdCase["id"].(string)
	if createdCase["status"] != "open" {
		t.Errorf("expected open, got %v", createdCase["status"])
	}

	// Resolve case with hide outcome.
	resolveBody, _ := json.Marshal(map[string]any{
		"status":  "resolved",
		"outcome": "hide",
		"notes":   "hidden by moderator",
	})
	resolveReq, _ := http.NewRequest(http.MethodPatch, server.URL+"/v1/moderation/cases/"+caseID, bytes.NewReader(resolveBody))
	resolveReq.Header.Set("Authorization", "Bearer "+staffToken)
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveReq.Header.Set("Idempotency-Key", "case-resolve-key-1")
	resolveResp, err := client.Do(resolveReq)
	if err != nil {
		t.Fatalf("resolve case: %v", err)
	}
	defer resolveResp.Body.Close()
	if resolveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d from resolve, got %d", http.StatusOK, resolveResp.StatusCode)
	}
	var resolved map[string]any
	if err := json.NewDecoder(resolveResp.Body).Decode(&resolved); err != nil {
		t.Fatalf("decode resolved case: %v", err)
	}
	if resolved["status"] != "resolved" || resolved["outcome"] != "hide" {
		t.Errorf("expected resolved/hide, got %v/%v", resolved["status"], resolved["outcome"])
	}

	// Post should no longer appear in public feed.
	feedResp, err := http.Get(server.URL + "/v1/posts")
	if err != nil {
		t.Fatalf("list posts after hide: %v", err)
	}
	defer feedResp.Body.Close()
	var feedPage map[string]any
	if err := json.NewDecoder(feedResp.Body).Decode(&feedPage); err != nil {
		t.Fatalf("decode feed: %v", err)
	}
	if len(feedPage["data"].([]any)) != 0 {
		t.Fatalf("expected 0 hidden posts in feed, got %d", len(feedPage["data"].([]any)))
	}

	// List case actions.
	actionsReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/moderation/cases/"+caseID+"/actions", nil)
	actionsReq.Header.Set("Authorization", "Bearer "+staffToken)
	actionsResp, err := client.Do(actionsReq)
	if err != nil {
		t.Fatalf("list actions: %v", err)
	}
	defer actionsResp.Body.Close()
	if actionsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d from actions, got %d", http.StatusOK, actionsResp.StatusCode)
	}
	var actions []map[string]any
	if err := json.NewDecoder(actionsResp.Body).Decode(&actions); err != nil {
		t.Fatalf("decode actions: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected at least one audit action")
	}
}

func TestHandler_StaffOnlyEndpointsRejectNonStaff(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountModerationHandler(f))
	defer server.Close()

	userToken := createSession(t, server.URL, "nonstaff", "password123")
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/moderation/cases", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list cases as user: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, resp.StatusCode)
	}
	var env map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if env["code"] != "MODERATION.NOT_MODERATOR" {
		t.Errorf("expected MODERATION.NOT_MODERATOR, got %v", env["code"])
	}
}
