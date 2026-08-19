package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yiguan/api/internal/auth"
)

func setupHandlerTest(t *testing.T) (*Handler, *auth.Handler, *recordingMailer) {
	cleanTables(t)
	cfg := newTestConfig()
	authRepo := auth.NewPostgresRepository(testDB)
	idRepo := NewPostgresRepository(testDB)
	mailer := &recordingMailer{}
	authLimiter := auth.NewMemoryLimiter()
	authSvc := auth.NewService(cfg, authRepo, mailer, authLimiter, &auth.StubTurnstile{})
	idLimiter := auth.NewMemoryLimiter()
	idSvc := NewService(cfg, idRepo, authRepo, mailer, idLimiter, nil)
	authHandler := auth.NewHandler(authSvc, cfg)
	idHandler := NewHandler(idSvc, authHandler, cfg)
	return idHandler, authHandler, mailer
}

func mountIdentityHandler(h *Handler, authHandler *auth.Handler) http.Handler {
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		authHandler.Mount(v1)
		h.Mount(v1)
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

func TestHandler_GetMe(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerme", "password123")

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var profile map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if _, ok := profile["emailMasked"]; !ok {
		t.Error("expected emailMasked key")
	}
	if profile["id"] == "" {
		t.Error("expected id")
	}
	if _, ok := profile["email"]; ok {
		t.Error("response must not contain raw email")
	}
}

func TestHandler_GetMe_Unauthorized(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/me")
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestHandler_CreateAndListPersona(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerpersona", "password123")

	body, _ := json.Marshal(map[string]any{"alias": "handlertest"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-persona-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created persona: %v", err)
	}
	if created["alias"] != "handlertest" {
		t.Errorf("expected alias handlertest, got %v", created["alias"])
	}

	// List should include the new persona.
	req2, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/me/personas", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("list personas: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	var list []map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 persona, got %d", len(list))
	}
}

func TestHandler_CreatePersona_MissingIdempotencyKey(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handleridemp", "password123")

	body, _ := json.Marshal(map[string]any{"alias": "missingkey"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope["code"] != "IDEMPOTENCY.MISSING_KEY" {
		t.Errorf("expected IDEMPOTENCY.MISSING_KEY, got %v", envelope["code"])
	}
}

func TestHandler_PublicPersona_NoRealProfileLeak(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerpublic", "password123")
	body, _ := json.Marshal(map[string]any{"alias": "publictest"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "public-persona-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created persona: %v", err)
	}
	personaID := created["id"].(string)

	// Public fetch without authentication.
	resp2, err := http.Get(server.URL + "/v1/personas/" + personaID)
	if err != nil {
		t.Fatalf("get public persona: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	var public map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&public); err != nil {
		t.Fatalf("decode public persona: %v", err)
	}
	if public["alias"] != "publictest" {
		t.Errorf("expected alias publictest, got %v", public["alias"])
	}
	for _, key := range []string{"realProfileId", "real_profile_id", "email", "emailMasked", "status", "isDefault", "updatedAt"} {
		if _, ok := public[key]; ok {
			t.Errorf("public persona response must not contain %s", key)
		}
	}
}

func TestHandler_PublicPersonaPosts_Empty(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerposts", "password123")
	body, _ := json.Marshal(map[string]any{"alias": "poststest"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "posts-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created persona: %v", err)
	}
	personaID := created["id"].(string)

	resp2, err := http.Get(server.URL + "/v1/personas/" + personaID + "/posts")
	if err != nil {
		t.Fatalf("get posts: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	var page map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&page); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	data, _ := page["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data, got %d", len(data))
	}
	pagination, _ := page["pagination"].(map[string]any)
	if pagination["hasMore"] != false {
		t.Error("expected hasMore false")
	}
}

func TestHandler_PublicPersonaPosts_Pagination(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerpostspage", "password123")
	body, _ := json.Marshal(map[string]any{"alias": "pagetest"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "posts-page-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created persona: %v", err)
	}
	personaID := created["id"].(string)

	resp2, err := http.Get(server.URL + "/v1/personas/" + personaID + "/posts?cursor=abc&limit=7")
	if err != nil {
		t.Fatalf("get posts: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp2.StatusCode)
	}

	var page map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&page); err != nil {
		t.Fatalf("decode posts: %v", err)
	}
	data, _ := page["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data, got %d", len(data))
	}
	pagination, _ := page["pagination"].(map[string]any)
	if pagination["hasMore"] != false {
		t.Error("expected hasMore false")
	}
	if pagination["nextCursor"] != nil {
		t.Errorf("expected nil nextCursor, got %v", pagination["nextCursor"])
	}
	if pagination["limit"] != float64(7) {
		t.Errorf("expected limit 7, got %v", pagination["limit"])
	}
}

func TestHandler_ArchivePersona(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerarchive", "password123")
	body, _ := json.Marshal(map[string]any{"alias": "archivetest"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "archive-create-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	defer resp.Body.Close()

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created persona: %v", err)
	}
	personaID := created["id"].(string)

	delReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/me/personas/"+personaID, nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delReq.Header.Set("Idempotency-Key", "archive-delete-key-1")
	delResp, err := client.Do(delReq)
	if err != nil {
		t.Fatalf("archive persona: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, delResp.StatusCode)
	}

	// Public endpoint should now 404.
	resp2, err := http.Get(server.URL + "/v1/personas/" + personaID)
	if err != nil {
		t.Fatalf("get public persona after archive: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected %d after archive, got %d", http.StatusNotFound, resp2.StatusCode)
	}
}

func TestHandler_SetDefaultPersona(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, "handlerdefault", "password123")
	client := &http.Client{Timeout: 5 * time.Second}

	create := func(alias, key string) string {
		body, _ := json.Marshal(map[string]any{"alias": alias})
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/me/personas", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("create %s: %v", alias, err)
		}
		defer resp.Body.Close()
		var created map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			t.Fatalf("decode %s: %v", alias, err)
		}
		return created["id"].(string)
	}

	first := create("first", "default-first-key")
	second := create("second", "default-second-key")

	putReq, _ := http.NewRequest(http.MethodPut, server.URL+"/v1/me/personas/"+second+"/default", nil)
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Idempotency-Key", "default-set-key")
	putResp, err := client.Do(putReq)
	if err != nil {
		t.Fatalf("set default: %v", err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, putResp.StatusCode)
	}

	var updated map[string]any
	if err := json.NewDecoder(putResp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated persona: %v", err)
	}
	if updated["id"] != second {
		t.Errorf("expected second persona, got %v", updated["id"])
	}
	if updated["isDefault"] != true {
		t.Error("expected second to be default")
	}

	// First should no longer be default.
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/me/personas/"+first, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("get first: %v", err)
	}
	defer resp.Body.Close()
	var firstAgain map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&firstAgain); err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if firstAgain["isDefault"] != false {
		t.Error("expected first to no longer be default")
	}
}

func TestHandler_ActivePersonaMiddleware_RequiresDefaultPersona(t *testing.T) {
	h, authHandler, _ := setupHandlerTest(t)
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		authHandler.Mount(v1)
		h.Mount(v1)
		v1.With(h.WithActivePersona()).Post("/test-active-persona", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
	server := httptest.NewServer(r)
	defer server.Close()

	token := createSession(t, server.URL, "handleractivepersona", "password123")

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/test-active-persona", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "active-persona-follow-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("active persona request: %v", err)
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
