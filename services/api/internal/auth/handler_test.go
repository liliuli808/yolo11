package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func setupHandlerTest(t *testing.T) (*Handler, *recordingMailer) {
	cleanTables(t)
	cfg := newTestConfig()
	repo := NewPostgresRepository(testDB)
	mailer := &recordingMailer{}
	limiter := NewMemoryLimiter()
	svc := NewService(cfg, repo, mailer, limiter, &StubTurnstile{})
	return NewHandler(svc, cfg), mailer
}

func mountHandler(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Route("/v1", func(v1 chi.Router) {
		h.Mount(v1)
	})
	return r
}

func registerSession(t *testing.T, h *Handler, serverURL, username, password string) sessionResponse {
	t.Helper()
	code := freshInviteCode(t, h.service)
	body, err := json.Marshal(map[string]any{"username": username, "password": password, "turnstileToken": "tok", "inviteCode": code})
	if err != nil {
		t.Fatalf("marshal register body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, serverURL+"/v1/auth/register", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build register request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "register-key-"+username)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post register: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var session sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.AccessToken == "" {
		t.Fatal("expected access token")
	}
	return session
}

func registerSessionToken(t *testing.T, svc *Service, username, password string) sessionResponse {
	t.Helper()
	code := freshInviteCode(t, svc)
	tokens, err := svc.Register(context.Background(), username, password, "tok", code, "127.0.0.1", "fp", "ua")
	if err != nil {
		t.Fatalf("register %s: %v", username, err)
	}
	return sessionResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		UserID:       tokens.UserID,
		PersonaID:    tokens.PersonaID,
		IsStaff:      tokens.IsStaff,
	}
}

func staffSession(t *testing.T, svc *Service) sessionResponse {
	t.Helper()
	staffID := ensureStaff(t, svc)
	tokens, err := svc.createSession(context.Background(), staffID, true, "127.0.0.1", "fp", "ua")
	if err != nil {
		t.Fatalf("create staff session: %v", err)
	}
	return sessionResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		UserID:       tokens.UserID,
		PersonaID:    tokens.PersonaID,
		IsStaff:      true,
	}
}

func bearerHeader(s sessionResponse) string {
	return "Bearer " + s.AccessToken
}

func TestHandler_Register(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	session := registerSession(t, h, server.URL, "hanna", "password123")
	if session.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if session.TokenType != "Bearer" {
		t.Errorf("expected Bearer, got %q", session.TokenType)
	}
	if session.ExpiresIn == 0 {
		t.Error("expected positive expires_in")
	}
	if session.UserID == "" {
		t.Error("expected user id")
	}
}

func TestHandler_Login(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	registerSession(t, h, server.URL, "harold", "password123")

	body, _ := json.Marshal(map[string]any{"username": "harold", "password": "password123"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "login-key-harold")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var session sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestHandler_Login_WrongPassword(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	registerSession(t, h, server.URL, "harriet", "password123")

	body, _ := json.Marshal(map[string]any{"username": "harriet", "password": "wrongpass"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "login-wrong-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope["code"] != "AUTH.INVALID_CREDENTIALS" {
		t.Errorf("expected AUTH.INVALID_CREDENTIALS, got %v", envelope["code"])
	}
}

func TestHandler_Register_UsernameTaken(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	registerSession(t, h, server.URL, "dave", "password123")

	code := freshInviteCode(t, h.service)
	body, _ := json.Marshal(map[string]any{"username": "Dave", "password": "password123", "turnstileToken": "tok", "inviteCode": code})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "register-key-dave-2")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post register: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, resp.StatusCode)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope["code"] != "AUTH.USERNAME_TAKEN" {
		t.Errorf("expected AUTH.USERNAME_TAKEN, got %v", envelope["code"])
	}
}

func TestHandler_RefreshSession(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	session := registerSessionToken(t, h.service, "helen", "password123")

	body, _ := json.Marshal(map[string]string{"refreshToken": session.RefreshToken})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "refresh-session-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post refresh: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var atr accessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&atr); err != nil {
		t.Fatalf("decode access token response: %v", err)
	}
	if atr.AccessToken == "" {
		t.Error("expected new access token")
	}

	found := false
	for _, c := range resp.Cookies() {
		if c.Name == refreshTokenCookie && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected rotated refresh token cookie")
	}
}

func TestHandler_DeleteSession(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	session := registerSessionToken(t, h.service, "hugo", "password123")

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Idempotency-Key", "delete-session-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, resp.StatusCode)
	}

	// Refreshing with the old token should now fail.
	body, _ := json.Marshal(map[string]string{"refreshToken": session.RefreshToken})
	req2, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/refresh", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "refresh-after-logout-key-1")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("post refresh after logout: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected %d after logout, got %d", http.StatusUnauthorized, resp2.StatusCode)
	}
}

func TestHandler_DeleteMe(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	session := registerSessionToken(t, h.service, "deleteme", "password123")

	body, _ := json.Marshal(map[string]string{"password": "password123"})
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "delete-me-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("delete me: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var state accountDeletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatalf("decode deletion state: %v", err)
	}
	if state.GracePeriodEndsAt == "" {
		t.Error("expected grace period end")
	}

	// Old refresh token should be rejected after deletion revokes sessions.
	body2, _ := json.Marshal(map[string]string{"refreshToken": session.RefreshToken})
	req2, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/refresh", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "refresh-after-deletion-key-1")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("post refresh after deletion: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected %d after deletion, got %d", http.StatusUnauthorized, resp2.StatusCode)
	}
}

func TestHandler_DeleteMe_AlreadyPending(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	ctx := context.Background()
	session := registerSessionToken(t, h.service, "deletepending", "password123")
	userID := UserIDFromToken(ctx, h.service, session.AccessToken)
	if _, err := h.service.RequestAccountDeletion(ctx, userID); err != nil {
		t.Fatalf("first deletion: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"password": "password123"})
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "delete-me-pending-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("delete me second: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, resp.StatusCode)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope["code"] != "ME.DELETION_ALREADY_PENDING" {
		t.Errorf("expected ME.DELETION_ALREADY_PENDING, got %v", envelope["code"])
	}
}

func TestHandler_DeleteMe_MissingIdempotencyKey(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	session := registerSessionToken(t, h.service, "deletemekey", "password123")

	body, _ := json.Marshal(map[string]string{"password": "password123"})
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/v1/me", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("delete me: %v", err)
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

func TestHandler_RateLimited_ReturnsContractCode(t *testing.T) {
	h, _ := setupHandlerTest(t)
	ctx := context.Background()

	// The register per-username limit is 5 per 10 minutes and is consumed
	// before the existing-user check. The first register succeeds, the next
	// four return ErrUsernameTaken but still consume a username-bucket slot.
	for i := 0; i < 5; i++ {
		code := freshInviteCode(t, h.service)
		_, err := h.service.Register(ctx, "ratelimited", "password123", "tok", code, "127.0.0.1", "fp1", "ua")
		if i == 0 {
			if err != nil {
				t.Fatalf("first register: expected success, got %v", err)
			}
			continue
		}
		if !errors.Is(err, ErrUsernameTaken) {
			t.Fatalf("register %d: expected ErrUsernameTaken, got %v", i+1, err)
		}
	}

	var rateLimitErr *RateLimitError
	code := freshInviteCode(t, h.service)
	if _, err := h.service.Register(ctx, "ratelimited", "password123", "tok", code, "127.0.0.1", "fp1", "ua"); !errors.As(err, &rateLimitErr) {
		t.Fatalf("expected rate limit error, got %v", err)
	}

	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	body, _ := json.Marshal(map[string]any{"username": "ratelimited", "password": "password123", "turnstileToken": "tok"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "rate-limit-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("post register: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}

	var envelope map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if envelope["code"] != "AUTH.RATE_LIMITED" {
		t.Errorf("expected AUTH.RATE_LIMITED, got %v", envelope["code"])
	}
	if resp.Header.Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}
