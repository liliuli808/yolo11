package content

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
		fmt.Println("TEST_DATABASE_URL or DATABASE_URL required for content tests")
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

	if err := resetContentSchema(ctx, pool); err != nil {
		fmt.Printf("reset content schema: %v\n", err)
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

func resetContentSchema(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`DROP TABLE IF EXISTS idempotency_keys`,
		`DROP TABLE IF EXISTS saves`,
		`DROP TABLE IF EXISTS media_assets`,
		`DROP TABLE IF EXISTS reactions`,
		`DROP TABLE IF EXISTS comments`,
		`DROP TABLE IF EXISTS posts`,
		`DROP TABLE IF EXISTS topic_follows`,
		`DROP TABLE IF EXISTS topics`,
		`DROP TABLE IF EXISTS moderation_actions`,
		`DROP TABLE IF EXISTS case_reports`,
		`DROP TABLE IF EXISTS moderation_cases`,
		`DROP TABLE IF EXISTS reports`,
		`DROP TABLE IF EXISTS blocks`,
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
	if _, err := testDB.Exec(ctx, "TRUNCATE TABLE idempotency_keys, saves, media_assets, reactions, comments, posts, topic_follows, topics, moderation_actions, case_reports, moderation_cases, reports, blocks, data_exports, personas, audit_events, sessions, email_codes, users RESTART IDENTITY CASCADE"); err != nil {
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
	authHandler    *auth.Handler
	idHandler      *identity.Handler
	contentHandler *Handler
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
	contentRepo := NewPostgresRepository(testDB)
	contentLimiter := auth.NewMemoryLimiter()
	contentSvc := NewService(cfg, contentRepo, idRepo, nil, contentLimiter)
	contentHandler := NewHandler(contentSvc, idHandler, idSvc, cfg)
	return &testFixtures{authHandler: authHandler, idHandler: idHandler, contentHandler: contentHandler}
}

func mountContentHandler(f *testFixtures) http.Handler {
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		f.authHandler.Mount(v1)
		f.idHandler.Mount(v1)
		f.contentHandler.Mount(v1)
	})
	return r
}

func createSession(t *testing.T, serverURL string, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"username": username, "password": password, "turnstileToken": "tok"})
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

func TestHandler_ListTopics(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/topics")
	if err != nil {
		t.Fatalf("list topics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatalf("decode topics: %v", err)
	}
	data := page["data"].([]any)
	if len(data) < 3 {
		t.Fatalf("expected at least 3 seeded topics, got %d", len(data))
	}
}

func TestHandler_CreatePost(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentcreatepost", "password123")
	_ = createPersona(t, server.URL, token, "contentposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "hello world"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-post-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var post map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	if post["content"] != "hello world" {
		t.Errorf("expected content hello world, got %v", post["content"])
	}
	if post["moderationState"] != "pendingReview" {
		t.Errorf("expected pendingReview, got %v", post["moderationState"])
	}
	topic := post["topic"].(map[string]any)
	if topic["id"] != topicID {
		t.Errorf("expected topic id %s, got %v", topicID, topic["id"])
	}
	if _, ok := post["realProfileId"]; ok {
		t.Error("post response must not contain realProfileId")
	}
}

func TestHandler_CreatePost_RequiresDefaultPersona(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentnopersona", "password123")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "hello world"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-post-no-persona-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
	var env map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if env["code"] != "PERSONA.DEFAULT_REQUIRED" {
		t.Errorf("expected PERSONA.DEFAULT_REQUIRED, got %v", env["code"])
	}
}

func TestHandler_FollowAndUnfollowTopic(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentfollow", "password123")
	_ = createPersona(t, server.URL, token, "follower")
	topicID := getTopicID(t, server.URL, "General")

	client := &http.Client{Timeout: 5 * time.Second}
	follow := func() *http.Response {
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/topics/"+topicID+"/follow", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Idempotency-Key", "follow-topic-key-1")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("follow topic: %v", err)
		}
		return resp
	}

	resp := follow()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Duplicate follow should conflict.
	resp2 := follow()
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, resp2.StatusCode)
	}

	// Unfollow.
	unfollowReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/topics/"+topicID+"/follow", nil)
	unfollowReq.Header.Set("Authorization", "Bearer "+token)
	resp3, err := client.Do(unfollowReq)
	if err != nil {
		t.Fatalf("unfollow topic: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, resp3.StatusCode)
	}
}

func TestHandler_ListTopicPosts(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contenttopicposts", "password123")
	_ = createPersona(t, server.URL, token, "topicposter")
	topicID := getTopicID(t, server.URL, "Reflection")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "topic post"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "topic-post-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	publishPost(t, created["id"].(string))

	resp2, err := http.Get(server.URL + "/v1/topics/" + topicID + "/posts")
	if err != nil {
		t.Fatalf("list topic posts: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp2.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&page); err != nil {
		t.Fatalf("decode topic posts: %v", err)
	}
	data := page["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 topic post, got %d", len(data))
	}
}

func TestHandler_UpdatePost(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentupdatepost", "password123")
	_ = createPersona(t, server.URL, token, "updateposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "before"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "update-post-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	postID := created["id"].(string)

	patchBody, _ := json.Marshal(map[string]any{"content": "after"})
	patchReq, _ := http.NewRequest(http.MethodPatch, server.URL+"/v1/posts/"+postID, bytes.NewReader(patchBody))
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Idempotency-Key", "update-post-patch-key-1")
	patchResp, err := client.Do(patchReq)
	if err != nil {
		t.Fatalf("update post: %v", err)
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, patchResp.StatusCode)
	}
	var updated map[string]any
	if err := json.NewDecoder(patchResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated post: %v", err)
	}
	if updated["content"] != "after" {
		t.Errorf("expected after, got %v", updated["content"])
	}
}

func TestHandler_CreateComment(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentcomment", "password123")
	_ = createPersona(t, server.URL, token, "commenter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "parent post"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "comment-post-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	postID := created["id"].(string)

	commentBody, _ := json.Marshal(map[string]any{"content": "nice post"})
	commentReq, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts/"+postID+"/comments", bytes.NewReader(commentBody))
	commentReq.Header.Set("Authorization", "Bearer "+token)
	commentReq.Header.Set("Content-Type", "application/json")
	commentReq.Header.Set("Idempotency-Key", "comment-create-key-1")
	commentResp, err := client.Do(commentReq)
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	defer commentResp.Body.Close()
	if commentResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, commentResp.StatusCode)
	}
	var comment map[string]any
	if err := json.NewDecoder(commentResp.Body).Decode(&comment); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	if comment["content"] != "nice post" {
		t.Errorf("expected nice post, got %v", comment["content"])
	}
}

func TestHandler_CreateReaction(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentreaction", "password123")
	_ = createPersona(t, server.URL, token, "reactor")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "reaction post"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "reaction-post-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	postID := created["id"].(string)

	reactBody, _ := json.Marshal(map[string]any{"type": "like"})
	reactReq, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts/"+postID+"/reactions", bytes.NewReader(reactBody))
	reactReq.Header.Set("Authorization", "Bearer "+token)
	reactReq.Header.Set("Content-Type", "application/json")
	reactReq.Header.Set("Idempotency-Key", "reaction-create-key-1")
	reactResp, err := client.Do(reactReq)
	if err != nil {
		t.Fatalf("create reaction: %v", err)
	}
	defer reactResp.Body.Close()
	if reactResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, reactResp.StatusCode)
	}
	var summary map[string]any
	if err := json.NewDecoder(reactResp.Body).Decode(&summary); err != nil {
		t.Fatalf("decode reaction summary: %v", err)
	}
	counts := summary["reactionCounts"].(map[string]any)
	if counts["like"] != float64(1) {
		t.Errorf("expected like count 1, got %v", counts["like"])
	}
}

func TestHandler_UploadMedia(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentmedia", "password123")
	_ = createPersona(t, server.URL, token, "mediauploader")

	body, _ := json.Marshal(map[string]any{
		"mimeType":  "image/png",
		"sizeBytes": 1024,
		"checksum":  "abc123",
	})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/media-uploads", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "media-upload-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("upload media: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var intent map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&intent); err != nil {
		t.Fatalf("decode media intent: %v", err)
	}
	if intent["assetId"] == "" {
		t.Error("expected assetId in media intent response")
	}
	if intent["uploadUrl"] == "" {
		t.Error("expected uploadUrl in media intent response")
	}
	if intent["status"] != "pending" {
		t.Errorf("expected pending status, got %v", intent["status"])
	}
}

func TestHandler_CreatePost_Idempotent(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentidempotent", "password123")
	_ = createPersona(t, server.URL, token, "idempotentposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "idempotent note"})
	idempotencyKey := "idempotent-create-key-1"
	client := &http.Client{Timeout: 5 * time.Second}

	create := func() (map[string]any, *http.Response) {
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("create post: %v", err)
		}
		var post map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
			resp.Body.Close()
			t.Fatalf("decode post: %v", err)
		}
		resp.Body.Close()
		return post, resp
	}

	post1, resp1 := create()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp1.StatusCode)
	}
	post2, resp2 := create()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("expected replay status %d, got %d", http.StatusCreated, resp2.StatusCode)
	}
	if post1["id"] != post2["id"] {
		t.Fatalf("idempotent replay returned different post: %v vs %v", post1["id"], post2["id"])
	}

	// A different idempotency key should create a second post.
	req2, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "idempotent-create-key-2")
	resp3, err := client.Do(req2)
	if err != nil {
		t.Fatalf("create second post: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp3.StatusCode)
	}
	var post3 map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&post3); err != nil {
		t.Fatalf("decode second post: %v", err)
	}
	if post3["id"] == post1["id"] {
		t.Fatal("different idempotency key returned the same post")
	}
}

func TestHandler_PostLifecycle(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentlifecycle", "password123")
	_ = createPersona(t, server.URL, token, "lifecycleposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "lifecycle note"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "lifecycle-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	postID := created["id"].(string)
	if created["moderationState"] != "pendingReview" {
		t.Fatalf("expected pendingReview, got %v", created["moderationState"])
	}

	// Author can see their own pending-review post.
	getReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/posts/"+postID, nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("get own post: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, getResp.StatusCode)
	}

	// Publish the post so it appears in feeds, then delete it.
	publishPost(t, postID)
	listResp, err := http.Get(server.URL + "/v1/posts")
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, listResp.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	if len(page["data"].([]any)) != 1 {
		t.Fatalf("expected 1 published post, got %d", len(page["data"].([]any)))
	}

	delReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/posts/"+postID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delReq.Header.Set("Idempotency-Key", "lifecycle-delete-key-1")
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("delete post: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, delResp.StatusCode)
	}

	// Public feed should now exclude the deleted post.
	listResp2, err := http.Get(server.URL + "/v1/posts")
	if err != nil {
		t.Fatalf("list posts after delete: %v", err)
	}
	defer listResp2.Body.Close()
	var page2 map[string]any
	if err := json.NewDecoder(listResp2.Body).Decode(&page2); err != nil {
		t.Fatalf("decode posts after delete: %v", err)
	}
	if len(page2["data"].([]any)) != 0 {
		t.Fatalf("expected 0 posts after delete, got %d", len(page2["data"].([]any)))
	}
}

func TestHandler_OptionalAuth_PopulatesViewerFields(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	posterToken := createSession(t, server.URL, "contentviewerposter", "password123")
	posterPersonaID := createPersona(t, server.URL, posterToken, "viewerposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "viewer fields note"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+posterToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "viewer-fields-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	postID := created["id"].(string)
	publishPost(t, postID)

	viewerToken := createSession(t, server.URL, "contentviewer", "password123")
	viewerPersonaID := createPersona(t, server.URL, viewerToken, "viewerpersona")

	// React and save the post as the viewer.
	reactBody, _ := json.Marshal(map[string]any{"type": "like"})
	reactReq, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts/"+postID+"/reactions", bytes.NewReader(reactBody))
	reactReq.Header.Set("Authorization", "Bearer "+viewerToken)
	reactReq.Header.Set("Content-Type", "application/json")
	reactReq.Header.Set("Idempotency-Key", "viewer-fields-react-key-1")
	reactResp, err := client.Do(reactReq)
	if err != nil {
		t.Fatalf("create reaction: %v", err)
	}
	reactResp.Body.Close()
	if reactResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d from reaction, got %d", http.StatusOK, reactResp.StatusCode)
	}

	saveReq, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts/"+postID+"/saves", nil)
	saveReq.Header.Set("Authorization", "Bearer "+viewerToken)
	saveReq.Header.Set("Idempotency-Key", "viewer-fields-save-key-1")
	saveResp, err := client.Do(saveReq)
	if err != nil {
		t.Fatalf("create save: %v", err)
	}
	saveResp.Body.Close()
	if saveResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected %d from save, got %d", http.StatusNoContent, saveResp.StatusCode)
	}

	// Unauthenticated request should not populate viewer fields.
	unauthResp, err := http.Get(server.URL + "/v1/posts/" + postID)
	if err != nil {
		t.Fatalf("get post unauthenticated: %v", err)
	}
	defer unauthResp.Body.Close()
	var unauthPost map[string]any
	if err := json.NewDecoder(unauthResp.Body).Decode(&unauthPost); err != nil {
		t.Fatalf("decode unauth post: %v", err)
	}
	if unauthPost["userReaction"] != nil {
		t.Errorf("expected no userReaction for unauthenticated request, got %v", unauthPost["userReaction"])
	}
	if unauthPost["isSaved"] != false {
		t.Errorf("expected isSaved false for unauthenticated request, got %v", unauthPost["isSaved"])
	}

	// Authenticated request as viewer should populate fields.
	authReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/posts/"+postID, nil)
	authReq.Header.Set("Authorization", "Bearer "+viewerToken)
	authResp, err := client.Do(authReq)
	if err != nil {
		t.Fatalf("get post authenticated: %v", err)
	}
	defer authResp.Body.Close()
	var authPost map[string]any
	if err := json.NewDecoder(authResp.Body).Decode(&authPost); err != nil {
		t.Fatalf("decode auth post: %v", err)
	}
	if authPost["userReaction"] != "like" {
		t.Errorf("expected userReaction like, got %v", authPost["userReaction"])
	}
	if authPost["isSaved"] != true {
		t.Errorf("expected isSaved true, got %v", authPost["isSaved"])
	}

	// Invalid token should return 401.
	badReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/posts/"+postID, nil)
	badReq.Header.Set("Authorization", "Bearer invalid-token")
	badResp, err := client.Do(badReq)
	if err != nil {
		t.Fatalf("get post invalid token: %v", err)
	}
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected %d for invalid token, got %d", http.StatusUnauthorized, badResp.StatusCode)
	}

	_ = posterPersonaID
	_ = viewerPersonaID
}

func TestHandler_FeedExcludesNonPublished(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentfeedexclude", "password123")
	_ = createPersona(t, server.URL, token, "feedposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "pending note"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "feed-exclude-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	// Public feed should not show pending-review posts.
	listResp, err := http.Get(server.URL + "/v1/posts")
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	defer listResp.Body.Close()
	var page map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&page); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	if len(page["data"].([]any)) != 0 {
		t.Fatalf("expected 0 pending posts in feed, got %d", len(page["data"].([]any)))
	}
}

func TestHandler_AuthorSeesOwnNonPublishedPosts(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentauthorstates", "password123")
	personaID := createPersona(t, server.URL, token, "stateposter")
	topicID := getTopicID(t, server.URL, "General")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "state note"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "author-states-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	postID := created["id"].(string)

	for _, state := range []string{"rejected", "hidden"} {
		ctx := context.Background()
		if _, err := testDB.Exec(ctx, "UPDATE posts SET moderation_state = $1 WHERE id = $2", state, postID); err != nil {
			t.Fatalf("set state %s: %v", state, err)
		}
		getReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/posts/"+postID, nil)
		getReq.Header.Set("Authorization", "Bearer "+token)
		getResp, err := client.Do(getReq)
		if err != nil {
			t.Fatalf("get own %s post: %v", state, err)
		}
		defer getResp.Body.Close()
		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("expected %d for own %s post, got %d", http.StatusOK, state, getResp.StatusCode)
		}
		var post map[string]any
		if err := json.NewDecoder(getResp.Body).Decode(&post); err != nil {
			t.Fatalf("decode %s post: %v", state, err)
		}
		if post["moderationState"] != state {
			t.Errorf("expected %s, got %v", state, post["moderationState"])
		}
	}

	// The author's persona posts feed should include the non-published note.
	feedReq, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/personas/"+personaID+"/posts", nil)
	feedReq.Header.Set("Authorization", "Bearer "+token)
	feedResp, err := client.Do(feedReq)
	if err != nil {
		t.Fatalf("list persona posts: %v", err)
	}
	defer feedResp.Body.Close()
	var feed map[string]any
	if err := json.NewDecoder(feedResp.Body).Decode(&feed); err != nil {
		t.Fatalf("decode persona posts: %v", err)
	}
	if len(feed["data"].([]any)) != 1 {
		t.Fatalf("expected 1 own post in persona feed, got %d", len(feed["data"].([]any)))
	}
}

func TestHandler_PublicPersonaPosts(t *testing.T) {
	f := setupHandlerTest(t)
	server := httptest.NewServer(mountContentHandler(f))
	defer server.Close()

	token := createSession(t, server.URL, "contentpersonaposts", "password123")
	personaID := createPersona(t, server.URL, token, "publicposter")
	topicID := getTopicID(t, server.URL, "Creative")

	body, _ := json.Marshal(map[string]any{"topicId": topicID, "content": "public post"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/posts", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "public-persona-post-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	publishPost(t, created["id"].(string))

	resp2, err := http.Get(server.URL + "/v1/personas/" + personaID + "/posts")
	if err != nil {
		t.Fatalf("list persona posts: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp2.StatusCode)
	}
	var page map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&page); err != nil {
		t.Fatalf("decode persona posts: %v", err)
	}
	data := page["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 persona post, got %d", len(data))
	}
}
