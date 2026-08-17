# Username/Password Auth + Cloudflare Turnstile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace email-code auth with username/password login/register across the Go API, OpenAPI contract, Android client, and admin web, protected by Cloudflare Turnstile on both endpoints.

**Architecture:** `users` table gains `username` (unique, case-insensitive) and `password_hash` (bcrypt). `POST /v1/auth/register` and `POST /v1/auth/login` replace `email-codes`/`email-sessions`; each validates a Turnstile token server-side via the Siteverify API. The Android client embeds Turnstile in a WebView; the admin web uses the official JS widget.

**Tech Stack:** Go 1.26 (chi, pgx, bcrypt via `golang.org/x/crypto`), PostgreSQL, OpenAPI 3.0, Jetpack Compose + Retrofit (Kotlin), React + Vite (admin).

---

## File Map

**Backend (`services/api`)**
- Create `internal/auth/turnstile.go` — TurnstileVerifier interface + Cloudflare impl
- Modify `internal/platform/config/config.go` — Turnstile keys, StaffUsernames
- Modify `internal/auth/service.go` — Register/Login/VerifyPasswordForDeletion, bcrypt, isStaff by username
- Modify `internal/auth/repository.go` — User.username/password_hash, CreateUser, GetUserByUsername, drop FindOrCreateUserByEmail
- Modify `internal/auth/handler.go` — register/login routes, DeleteMe password, error mapping
- Modify `internal/auth/limiter.go` — username rate-limit key builder
- Modify `cmd/api/main.go` — wire CloudflareTurnstileVerifier
- Modify `migrations/auth/003_username_password.up.sql` / `.down.sql` — new migration
- Modify `infra/.env.example` — Turnstile keys, STAFF_USERNAMES
- Modify tests: `internal/auth/service_test.go`, `internal/auth/handler_test.go`, and the `createSession` helpers + auth-service constructors in `internal/content/handler_test.go`, `internal/content/service_test.go`, `internal/identity/handler_test.go`, `internal/identity/service_test.go`, `internal/moderation/handler_test.go`

**Contract**
- Modify `contracts/openapi/openapi.yaml` — register/login endpoints, requests, Session.userId, DELETE /v1/me password

**Android (`apps/android`)**
- Create `app/src/main/java/app/rebuild/social/core/design/components/TurnstileWebView.kt`
- Create `app/src/main/java/app/rebuild/social/feature/auth/LoginScreen.kt`
- Create `app/src/main/java/app/rebuild/social/feature/auth/RegisterScreen.kt`
- Create `app/src/main/java/app/rebuild/social/feature/auth/AuthViewModel.kt`
- Delete `app/src/main/java/app/rebuild/social/feature/auth/EmailSignInScreen.kt`, `VerificationScreen.kt`, `VerificationCodeInput.kt`
- Modify `core/network/ApiClient.kt`, `core/network/LanternApiClient.kt`, `core/network/NetworkModule.kt` (site key)
- Modify `navigation/Routes.kt`, `navigation/RootNavigation.kt`
- Modify `app/build.gradle.kts` — TURNSTILE_SITE_KEY BuildConfig
- Modify `androidTest/.../navigation/RootNavigationTest.kt`

**Admin (`apps/admin`)**
- Modify `src/api/client.ts`, `src/pages/Login.tsx`, `src/types.ts`

---

## Phase 1: Go backend

### Task 1: Add bcrypt dependency

**Files:**
- Modify: `services/api/go.mod`, `services/api/go.sum`

- [ ] **Step 1: Add the dependency**

Run:
```bash
cd services/api && go get golang.org/x/crypto@latest
```
Expected: `go.mod` gains `golang.org/x/crypto vX.Y.Z` under `require` (direct), `go.sum` updated.

- [ ] **Step 2: Verify build still works**

Run:
```bash
cd services/api && go build ./...
```
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add services/api/go.mod services/api/go.sum
git commit -m "feat(auth): add golang.org/x/crypto for bcrypt"
```

---

### Task 2: Migration 003 — username + password_hash

**Files:**
- Create: `services/api/migrations/auth/003_username_password.up.sql`
- Create: `services/api/migrations/auth/003_username_password.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
ALTER TABLE users ADD COLUMN username TEXT;
ALTER TABLE users ADD COLUMN password_hash TEXT;

CREATE UNIQUE INDEX idx_users_username ON users (username);

ALTER TABLE users ALTER COLUMN email_normalized DROP NOT NULL;
```

- [ ] **Step 2: Write the down migration**

```sql
ALTER TABLE users ALTER COLUMN email_normalized SET NOT NULL;
DROP INDEX IF EXISTS idx_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
ALTER TABLE users DROP COLUMN IF EXISTS username;
```

- [ ] **Step 3: Commit**

```bash
git add services/api/migrations/auth/003_username_password.up.sql services/api/migrations/auth/003_username_password.down.sql
git commit -m "feat(auth): add username and password_hash to users"
```

---

### Task 3: Config — Turnstile keys + staff usernames

**Files:**
- Modify: `services/api/internal/platform/config/config.go:14-43`

- [ ] **Step 1: Add config fields**

Replace the struct fields section with:

```go
	EmailCodeKey    string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	EmailAdapter    string
	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPTLS         bool
	StaffUsernames  []string
	AdminSessionTTL time.Duration

	TurnstileSecretKey        string
	TurnstileSiteKey          string
	TurnstileExpectedHostname string
```

- [ ] **Step 2: Add to required env list**

In `Load`, after the `{"EMAIL_CODE_KEY", ...}` line:

```go
		{"TURNSTILE_SECRET_KEY", state.getenv("TURNSTILE_SECRET_KEY")},
```

- [ ] **Step 3: Populate the config values**

In the `cfg := &Config{...}` literal, replace the `StaffEmails: parseStringList(state.getenv("STAFF_EMAILS")),` line with:

```go
		StaffUsernames:           parseStringList(state.getenv("STAFF_USERNAMES")),
		TurnstileSecretKey:       required[?].value, // see note below
		TurnstileSiteKey:         defaultString(state.getenv("TURNSTILE_SITE_KEY"), ""),
		TurnstileExpectedHostname: defaultString(state.getenv("TURNSTILE_EXPECTED_HOSTNAME"), ""),
```

Note: because `TurnstileSecretKey` was appended to the required slice, it becomes `required[6]` (the email-code key was index 5). Update the literal accordingly: `TurnstileSecretKey: required[6].value`. Verify the final index by reading the final file.

- [ ] **Step 4: Update `.env.example`**

Modify `infra/.env.example`:

```
# Optional: comma-separated list of staff usernames. Staff members receive
# an `isStaff` claim in their access token and can access moderation endpoints.
STAFF_USERNAMES=admin

# Cloudflare Turnstile (https://developers.cloudflare.com/turnstile/)
# Test secret (always passes): 1x0000000000000000000000000000000AA
# Test site key (always passes): 1x00000000000000000000AA
TURNSTILE_SECRET_KEY=1x0000000000000000000000000000000AA
TURNSTILE_SITE_KEY=1x00000000000000000000AA
TURNSTILE_EXPECTED_HOSTNAME=
```

Also remove the old `STAFF_EMAILS=staff@example.com` block.

- [ ] **Step 5: Update the config test**

Modify `services/api/internal/platform/config/config_test.go` to cover the new required key and the `StaffUsernames`/Turnstile parsing. Follow the file's existing table style. At minimum assert that loading with `TURNSTILE_SECRET_KEY` set succeeds, and that `STAFF_USERNAMES=a,b` parses to `[]string{"a","b"}`.

- [ ] **Step 6: Run tests**

Run:
```bash
cd services/api && TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/platform/config/... -count=1
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add services/api/internal/platform/config/config.go services/api/internal/platform/config/config_test.go infra/.env.example
git commit -m "feat(auth): add turnstile config and staff usernames"
```

---

### Task 4: Turnstile verifier

**Files:**
- Create: `services/api/internal/auth/turnstile.go`

- [ ] **Step 1: Write the Turnstile verifier**

```go
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yiguan/api/internal/platform/config"
)

// TurnstileVerifier validates a Cloudflare Turnstile token.
type TurnstileVerifier interface {
	Verify(ctx context.Context, token string) error
}

// CloudflareTurnstileVerifier verifies tokens against the Siteverify API.
type CloudflareTurnstileVerifier struct {
	secretKey string
	hostname  string
	client    *http.Client
}

// NewCloudflareTurnstileVerifier creates a verifier from configuration.
func NewCloudflareTurnstileVerifier(cfg *config.Config) *CloudflareTurnstileVerifier {
	return &CloudflareTurnstileVerifier{
		secretKey: cfg.TurnstileSecretKey,
		hostname:  cfg.TurnstileExpectedHostname,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Verify validates token. Empty tokens always fail.
func (v *CloudflareTurnstileVerifier) Verify(ctx context.Context, token string) error {
	if token == "" {
		return ErrCaptchaFailed
	}
	payload, err := json.Marshal(map[string]string{
		"secret":   v.secretKey,
		"response": token,
	})
	if err != nil {
		return fmt.Errorf("encode siteverify payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("call siteverify: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read siteverify response: %w", err)
	}

	var result struct {
		Success  bool     `json:"success"`
		Hostname string   `json:"hostname"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode siteverify response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("%w: %v", ErrCaptchaFailed, result.ErrorCodes)
	}
	if v.hostname != "" && result.Hostname != v.hostname {
		return ErrCaptchaFailed
	}
	return nil
}
```

- [ ] **Step 2: Add the domain error**

In `services/api/internal/auth/service.go`, add to the `var (...)` error block:

```go
	ErrUsernameTaken       = errors.New("username already taken")
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrCaptchaFailed       = errors.New("captcha verification failed")
	ErrInvalidUsername     = errors.New("invalid username")
	ErrInvalidPassword     = errors.New("invalid password")
```

- [ ] **Step 3: Verify build**

Run:
```bash
cd services/api && go build ./...
```
Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add services/api/internal/auth/turnstile.go services/api/internal/auth/service.go
git commit -m "feat(auth): add cloudflare turnstile verifier"
```

---

### Task 5: Repository — username/password persistence

**Files:**
- Modify: `services/api/internal/auth/repository.go`

- [ ] **Step 1: Add fields to the User struct**

Replace the `User` struct with:

```go
// User is the authenticated account root used by the auth subsystem.
type User struct {
	ID                        string
	Username                  string
	PasswordHash              string
	EmailNormalized           string
	Status                    string
	DeletionRequestedAt       *time.Time
	DeletionGracePeriodEndsAt *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}
```

- [ ] **Step 2: Update the interface**

In the `Repository` interface, replace `FindOrCreateUserByEmail` with:

```go
	CreateUser(ctx context.Context, username, passwordHash string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
```

Keep `GetUserByID` and all deletion methods.

- [ ] **Step 3: Update the shared user row scanner**

Add a helper that scans the full user row and use it everywhere `users` columns are read. Replace `FindOrCreateUserByEmail` with:

```go
const userColumns = `id, username, password_hash, email_normalized, status, deletion_requested_at, deletion_grace_period_ends_at, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	if err := row.Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.EmailNormalized,
		&u.Status,
		&u.DeletionRequestedAt,
		&u.DeletionGracePeriodEndsAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, username, passwordHash string) (*User, error) {
	const sql = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING ` + userColumns
	return scanUser(r.pool.QueryRow(ctx, sql, username, passwordHash))
}

func (r *PostgresRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	const sql = `
		SELECT ` + userColumns + `
		FROM users
		WHERE username = $1
	`
	u, err := scanUser(r.pool.QueryRow(ctx, sql, username))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select user by username: %w", err)
	}
	return u, nil
}
```

- [ ] **Step 4: Update the remaining user scans**

Modify `GetUserByID`, `ListUsersPastGracePeriod`, and `MarkUserDeleting` (RETURNING scan) to select the new column set (`userColumns` for SELECTs; for `MarkUserDeleting` keep `RETURNING status`). `MarkUserDeleted` keeps `RETURNING status`. Use `scanUser` in `GetUserByID` and `ListUsersPastGracePeriod` loops.

- [ ] **Step 5: Verify build**

Run:
```bash
cd services/api && go build ./...
```
Expected: no output, exit 0.

- [ ] **Step 6: Commit**

```bash
git add services/api/internal/auth/repository.go
git commit -m "feat(auth): add username/password repository methods"
```

---

### Task 6: Rate-limiter username key

**Files:**
- Modify: `services/api/internal/auth/limiter.go:106-119`

- [ ] **Step 1: Add a username key builder**

Add to `keyBuilder`:

```go
func (keyBuilder) username(prefix, username string) string {
	return fmt.Sprintf("%s:username:%s", prefix, username)
}
```

- [ ] **Step 2: Commit**

```bash
git add services/api/internal/auth/limiter.go
git commit -m "feat(auth): add username rate-limit key builder"
```

---

### Task 7: Service — Register / Login / password deletion

**Files:**
- Modify: `services/api/internal/auth/service.go`

- [ ] **Step 1: Add turnstile to the Service**

Add a field to the `Service` struct and to `NewService`:

```go
type Service struct {
	cfg         *config.Config
	repo        Repository
	mailer      Mailer
	limiter     RateLimiter
	turnstile   TurnstileVerifier
	signer      []byte
	codeKey     []byte
	keys        keyBuilder
	IdentityCleanup func(ctx context.Context, userID string) error
}
```

```go
func NewService(cfg *config.Config, repo Repository, mailer Mailer, limiter RateLimiter, turnstile TurnstileVerifier) *Service {
	return &Service{
		cfg:       cfg,
		repo:      repo,
		mailer:    mailer,
		limiter:   limiter,
		turnstile: turnstile,
		signer:    []byte(cfg.JWTSigningKey),
		codeKey:   []byte(cfg.EmailCodeKey),
	}
}
```

- [ ] **Step 2: Add validation helpers**

Add near the bottom of the file:

```go
var usernameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,19}$`)

// NormalizeUsername lowercases and trims a username.
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func validateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	return nil
}
```

Add `"regexp"` to the imports.

- [ ] **Step 3: Add rate limits**

Add to the rate-limit var block:

```go
	registerByUsername = RateLimit{Count: 5, Window: 10 * time.Minute}
	registerByIP       = RateLimit{Count: 20, Window: 10 * time.Minute}
	registerByFP       = RateLimit{Count: 10, Window: 10 * time.Minute}

	loginByUsername = RateLimit{Count: 10, Window: 10 * time.Minute}
	loginByIP       = RateLimit{Count: 40, Window: 10 * time.Minute}
	loginByFP       = RateLimit{Count: 20, Window: 10 * time.Minute}
```

- [ ] **Step 4: Add Register and Login methods**

```go
// Register creates a user with username/password and a session.
func (s *Service) Register(ctx context.Context, username, password, turnstileToken, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	if err := s.verifyTurnstile(ctx, turnstileToken); err != nil {
		return nil, err
	}
	normalized := NormalizeUsername(username)
	if err := validateUsername(normalized); err != nil {
		return nil, ErrInvalidUsername
	}
	if err := validatePassword(password); err != nil {
		return nil, ErrInvalidPassword
	}

	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.username("register", normalized), registerByUsername); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.ip("register", ip), registerByIP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.fingerprint("register", fingerprint), registerByFP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	existing, err := s.repo.GetUserByUsername(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	tokens, err := s.createSession(ctx, user.ID, s.isStaff(user.Username), ip, fingerprint, userAgent)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &user.ID, &tokens.SessionID, "account.registered", ip, userAgent, fingerprint, nil)
	return tokens, nil
}

// Login validates credentials and creates a session.
func (s *Service) Login(ctx context.Context, username, password, turnstileToken, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	if err := s.verifyTurnstile(ctx, turnstileToken); err != nil {
		return nil, err
	}
	normalized := NormalizeUsername(username)
	if err := validateUsername(normalized); err != nil {
		return nil, ErrInvalidCredentials
	}
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.username("login", normalized), loginByUsername); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.ip("login", ip), loginByIP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.fingerprint("login", fingerprint), loginByFP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	user, err := s.repo.GetUserByUsername(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.createSession(ctx, user.ID, s.isStaff(user.Username), ip, fingerprint, userAgent)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &user.ID, &tokens.SessionID, "session.created", ip, userAgent, fingerprint, map[string]any{"is_staff": s.isStaff(user.Username)})
	return tokens, nil
}
```

- [ ] **Step 5: Extract a session-creation helper**

Add:

```go
// createSession builds a session for the user and returns access + refresh tokens.
func (s *Service) createSession(ctx context.Context, userID string, isStaff bool, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	refreshToken, refreshHash, err := s.newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	session := &Session{
		UserID:           userID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        now.Add(s.cfg.RefreshTokenTTL),
		IPAddress:        ip,
		UserAgent:        userAgent,
		Fingerprint:      fingerprint,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	accessToken, expiresIn, err := s.issueAccessToken(userID, session.ID, isStaff)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		IsStaff:      isStaff,
	}, nil
}
```

Note: `TokenResponse` gains a `SessionID` field. Add `SessionID string` to the `TokenResponse` struct so `Register` can audit it.

- [ ] **Step 6: Add VerifyPasswordForDeletion**

```go
// VerifyPasswordForDeletion checks the account password to confirm deletion.
func (s *Service) VerifyPasswordForDeletion(ctx context.Context, userID, password string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUserNotFound, err)
	}
	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return ErrDeletionInvalidPassword
	}
	return nil
}
```

- [ ] **Step 7: Update isStaff to use usernames**

Replace `isStaff` with:

```go
func (s *Service) isStaff(username string) bool {
	if len(s.cfg.StaffUsernames) == 0 {
		return false
	}
	normalized := NormalizeUsername(username)
	for _, u := range s.cfg.StaffUsernames {
		if NormalizeUsername(u) == normalized {
			return true
		}
	}
	return false
}
```

- [ ] **Step 8: Add verifyTurnstile helper**

```go
func (s *Service) verifyTurnstile(ctx context.Context, token string) error {
	if s.turnstile == nil {
		return nil
	}
	return s.turnstile.Verify(ctx, token)
}
```

- [ ] **Step 9: Fix the remaining email-code references**

`RequestEmailCode`, `CreateEmailSession`, and `VerifyEmailCode` still reference `FindOrCreateUserByEmail`. These methods are removed by the handler task; for now the file must still compile. Temporarily keep `FindOrCreateUserByEmail` compile errors unresolved ONLY if the build fails — instead, leave these three methods in place until Task 9 (handler) removes their callers, then remove the dead methods. `CreateEmailSession`/`RequestEmailCode` may be deleted in this task; do not leave unused code. The `identity` package still imports `auth` for `EmailCode`/`CreateEmailCode`/`GetActiveEmailCode` — keep the email-code repository methods and `VerifyEmailCode` if identity still calls it via its own copy (it does not; identity has its own `verifyEmailCode`). Remove `RequestEmailCode`, `CreateEmailSession`, and their rate-limit constants; keep `VerifyEmailCode` only if referenced — otherwise remove it too.

- [ ] **Step 10: Update the service constructor call in main**

Run:
```bash
cd services/api && go build ./...
```
Expected: compile errors in `cmd/api/main.go` (missing turnstile arg) and in test files. Fix `cmd/api/main.go` in Task 10; tests in Task 11. For now, confirm only those call sites fail.

- [ ] **Step 11: Commit**

```bash
git add services/api/internal/auth/service.go
git commit -m "feat(auth): implement register/login with bcrypt and turnstile"
```

---

### Task 8: Handler — register/login routes, DeleteMe password

**Files:**
- Modify: `services/api/internal/auth/handler.go`

- [ ] **Step 1: Replace the route mounts**

In `Mount`, replace the two email routes:

```go
	r.Post("/auth/register", h.Register)
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.RefreshSession)
```

- [ ] **Step 2: Add request/response types**

Replace `emailCodeRequest` and `emailSessionRequest` with:

```go
type registerRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	TurnstileToken string `json:"turnstileToken"`
}

type loginRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	TurnstileToken string `json:"turnstileToken"`
}
```

- [ ] **Step 3: Add Register and Login handlers**

Replace `SendEmailCode` and `CreateEmailSession` with:

```go
// Register handles POST /v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req registerRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	ip := clientIP(r, h.cfg.RateLimitBehindProxy)
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	tokens, err := h.service.Register(r.Context(), req.Username, req.Password, req.TurnstileToken, ip, fingerprint, r.UserAgent())
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken, int(h.cfg.RefreshTokenTTL.Seconds()))

	if err := httpx.WriteJSON(w, http.StatusCreated, h.sessionResponse(tokens)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// Login handles POST /v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req loginRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	ip := clientIP(r, h.cfg.RateLimitBehindProxy)
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	tokens, err := h.service.Login(r.Context(), req.Username, req.Password, req.TurnstileToken, ip, fingerprint, r.UserAgent())
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken, int(h.cfg.RefreshTokenTTL.Seconds()))

	if err := httpx.WriteJSON(w, http.StatusOK, h.sessionResponse(tokens)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}
```

- [ ] **Step 4: Add a sessionResponse builder**

```go
func (h *Handler) sessionResponse(tokens *auth.TokenResponse) sessionResponse {
	return sessionResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		PersonaID:    nil,
		IsStaff:      tokens.IsStaff,
	}
}
```

(The handler already imports `github.com/yiguan/api/internal/auth` — verify; if not, add it.)

- [ ] **Step 5: Update DeleteMe to use password**

Replace the code-block in `DeleteMe` that calls `VerifyEmailCode`:

```go
	if err := h.service.VerifyPasswordForDeletion(r.Context(), userID, req.Password); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
```

Update the `accountDeletionRequest` struct:

```go
type accountDeletionRequest struct {
	Password string `json:"password"`
}
```

- [ ] **Step 6: Add error mappings**

In `respondDomainError`, add cases before the `default`:

```go
	case errors.Is(err, ErrInvalidUsername):
		httpError(ctx, w, http.StatusBadRequest, "AUTH.INVALID_USERNAME", "username must be 3-20 letters, digits, or underscores")
	case errors.Is(err, ErrInvalidPassword):
		httpError(ctx, w, http.StatusBadRequest, "AUTH.INVALID_PASSWORD", "password must be at least 8 characters")
	case errors.Is(err, ErrUsernameTaken):
		httpError(ctx, w, http.StatusConflict, "AUTH.USERNAME_TAKEN", "username is already taken")
	case errors.Is(err, ErrInvalidCredentials):
		httpError(ctx, w, http.StatusUnauthorized, "AUTH.INVALID_CREDENTIALS", "invalid username or password")
	case errors.Is(err, ErrCaptchaFailed):
		httpError(ctx, w, http.StatusBadRequest, "AUTH.CAPTCHA_FAILED", "human verification failed, please try again")
```

- [ ] **Step 7: Verify build**

Run:
```bash
cd services/api && go build ./...
```
Expected: only `cmd/api/main.go` and test files still fail to compile.

- [ ] **Step 8: Commit**

```bash
git add services/api/internal/auth/handler.go
git commit -m "feat(auth): add register/login handlers, password deletion"
```

---

### Task 9: Wire the verifier in main.go

**Files:**
- Modify: `services/api/cmd/api/main.go:70-74`

- [ ] **Step 1: Create the turnstile verifier**

Replace the auth-service construction block:

```go
	authRepo := auth.NewPostgresRepository(pool)
	authMailer := auth.NewMailerFromConfig(cfg, logger)
	authLimiter := auth.NewMemoryLimiter()
	turnstileVerifier := auth.NewCloudflareTurnstileVerifier(cfg)
	authService := auth.NewService(cfg, authRepo, authMailer, authLimiter, turnstileVerifier)
	authHandler := auth.NewHandler(authService, cfg)
```

- [ ] **Step 2: Verify build**

Run:
```bash
cd services/api && go build ./...
```
Expected: no output, exit 0 (test files excluded).

- [ ] **Step 3: Commit**

```bash
git add services/api/cmd/api/main.go
git commit -m "feat(auth): wire turnstile verifier into api server"
```

---

### Task 10: Rewrite auth service tests

**Files:**
- Modify: `services/api/internal/auth/service_test.go`

- [ ] **Step 1: Add a stub turnstile and update setup helpers**

Replace `newTestConfig`, `newTestService`, and `setupHandlerTest` construction of `NewService` with a turnstile arg. Add:

```go
type stubTurnstile struct {
	fail bool
}

func (s *stubTurnstile) Verify(ctx context.Context, token string) error {
	if s.fail {
		return ErrCaptchaFailed
	}
	return nil
}
```

Update `newTestConfig` to include `StaffUsernames: []string{"admin"}`.

Update `newTestService` and the service constructor calls to:

```go
return NewService(newTestConfig(), repo, mailer, limiter, &stubTurnstile{}), mailer
```

- [ ] **Step 2: Replace the email-code tests**

Delete `TestCreateEmailSession_Success`, `TestCreateEmailSession_CodeReuseFails`, `TestCreateEmailSession_ExpiredCodeFails`, `TestRequestEmailCode_RateLimitedByEmail`, and the other `RequestEmailCode`/`CreateEmailSession` tests. Add:

```go
func TestRegister_Success(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()

	tokens, err := svc.Register(context.Background(), "Alice_1", "password123", "tok", "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("expected access and refresh tokens")
	}

	user, err := svc.repo.GetUserByUsername(context.Background(), "alice_1")
	if err != nil || user == nil {
		t.Fatalf("expected user to exist, got %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")) != nil {
		t.Error("password hash does not match")
	}
}

func TestRegister_UsernameTaken(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	if _, err := svc.Register(context.Background(), "bob", "password123", "tok", "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(context.Background(), "Bob", "password123", "tok", "127.0.0.1", "fp2", "ua")
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestRegister_CaptchaFails(t *testing.T) {
	cleanTables(t)
	svc := NewService(newTestConfig(), NewPostgresRepository(testDB), &recordingMailer{}, NewMemoryLimiter(), &stubTurnstile{fail: true})
	_, err := svc.Register(context.Background(), "carol", "password123", "tok", "127.0.0.1", "fp1", "ua")
	if !errors.Is(err, ErrCaptchaFailed) {
		t.Fatalf("expected ErrCaptchaFailed, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	if _, err := svc.Register(context.Background(), "dave", "password123", "tok", "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("register: %v", err)
	}
	tokens, err := svc.Login(context.Background(), "dave", "password123", "tok", "127.0.0.1", "fp2", "ua")
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
	if _, err := svc.Register(context.Background(), "erin", "password123", "tok", "127.0.0.1", "fp1", "ua"); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err := svc.Login(context.Background(), "erin", "wrongpass", "tok", "127.0.0.1", "fp2", "ua")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	_, err := svc.Login(context.Background(), "nobody", "password123", "tok", "127.0.0.1", "fp1", "ua")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestVerifyPasswordForDeletion(t *testing.T) {
	cleanTables(t)
	svc, _ := newTestService()
	tokens, err := svc.Register(context.Background(), "frank", "password123", "tok", "127.0.0.1", "fp1", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.VerifyPasswordForDeletion(context.Background(), tokens.UserID, "password123"); err != nil {
		t.Fatalf("expected valid password, got %v", err)
	}
	if err := svc.VerifyPasswordForDeletion(context.Background(), tokens.UserID, "wrong"); err == nil {
		t.Fatal("expected error for wrong password")
	}
}
```

Note: `TokenResponse` must expose `UserID` (add `UserID string` in the Register method by assigning `UserID: user.ID`).

- [ ] **Step 3: Update imports**

Add `golang.org/x/crypto/bcrypt` to the test imports. Keep `errors`, `context`, `testing` (already present).

- [ ] **Step 4: Run the auth service tests**

Run:
```bash
cd services/api && TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/auth/ -run 'TestRegister|TestLogin|TestVerifyPassword' -count=1
```
Expected: PASS. (Requires `make migrate-test-up` first if the test DB lacks migration 003.)

- [ ] **Step 5: Run the full auth package**

Run:
```bash
cd services/api && TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/auth/... -count=1
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/api/internal/auth/service_test.go services/api/internal/auth/service.go
git commit -m "test(auth): register/login service tests"
```

---

### Task 11: Rewrite auth handler tests

**Files:**
- Modify: `services/api/internal/auth/handler_test.go`

- [ ] **Step 1: Update setup**

Update `setupHandlerTest` to pass the turnstile:

```go
func setupHandlerTest(t *testing.T) (*Handler, *recordingMailer) {
	cleanTables(t)
	cfg := newTestConfig()
	repo := NewPostgresRepository(testDB)
	mailer := &recordingMailer{}
	limiter := NewMemoryLimiter()
	svc := NewService(cfg, repo, mailer, limiter, &stubTurnstile{})
	return NewHandler(svc, cfg), mailer
}
```

- [ ] **Step 2: Replace the email-code tests**

Delete `TestHandler_SendEmailCode`, `TestHandler_CreateEmailSession`, `TestHandler_CreateEmailSession_InvalidCode`, and the refresh/logout/delete tests that call `RequestEmailCode`/`CreateEmailSession`. Replace the session-creation flow with a helper:

```go
func registerSession(t *testing.T, serverURL, username, password string) sessionResponse {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"username": username, "password": password, "turnstileToken": "tok"})
	req, _ := http.NewRequest(http.MethodPost, serverURL+"/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "register-key-"+username)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var session sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return session
}
```

Add handler tests:

```go
func TestHandler_Register(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	body, _ := json.Marshal(map[string]string{"username": "alice", "password": "password123", "turnstileToken": "tok"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "register-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestHandler_Login(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	registerSession(t, server.URL, "bob", "password123")

	body, _ := json.Marshal(map[string]string{"username": "bob", "password": "password123", "turnstileToken": "tok"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "login-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandler_Login_WrongPassword(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	registerSession(t, server.URL, "carol", "password123")

	body, _ := json.Marshal(map[string]string{"username": "carol", "password": "wrongpass", "turnstileToken": "tok"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "login-key-2")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandler_Register_UsernameTaken(t *testing.T) {
	h, _ := setupHandlerTest(t)
	server := httptest.NewServer(mountHandler(h))
	defer server.Close()

	registerSession(t, server.URL, "dave", "password123")

	body, _ := json.Marshal(map[string]string{"username": "Dave", "password": "password123", "turnstileToken": "tok"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "register-key-2")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 3: Update refresh/logout/deletion tests**

For the remaining refresh/logout/DeleteMe tests, replace `RequestEmailCode`+`CreateEmailSession` setup with a direct service register:

```go
func registerSessionToken(t *testing.T, svc *Service, username, password string) sessionResponse {
	t.Helper()
	tokens, err := svc.Register(context.Background(), username, password, "tok", "127.0.0.1", "fp", "ua")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return sessionResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, TokenType: tokens.TokenType, ExpiresIn: tokens.ExpiresIn, IsStaff: tokens.IsStaff}
}
```

For the DeleteMe test, register the user, then POST `DELETE /v1/me` with `{"password":"password123"}` and expect 200.

- [ ] **Step 4: Run handler tests**

Run:
```bash
cd services/api && TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/auth/ -run TestHandler -count=1
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/api/internal/auth/handler_test.go
git commit -m "test(auth): register/login handler tests"
```

---

### Task 12: Update cross-module tests and helpers

**Files:**
- Modify: `services/api/internal/content/handler_test.go`, `services/api/internal/content/service_test.go`
- Modify: `services/api/internal/identity/handler_test.go`, `services/api/internal/identity/service_test.go`
- Modify: `services/api/internal/moderation/handler_test.go`

- [ ] **Step 1: Update auth-service constructors**

Everywhere these tests call `auth.NewService(...)`, append `, &auth.StubTurnstile{}` as the last argument. Export a stub from the auth package so other packages can reuse it. Add to `services/api/internal/auth/turnstile.go`:

```go
// StubTurnstile is a test double that always passes unless Fail is set.
type StubTurnstile struct {
	Fail bool
}

// Verify implements TurnstileVerifier.
func (s *StubTurnstile) Verify(ctx context.Context, token string) error {
	if s.Fail {
		return ErrCaptchaFailed
	}
	return nil
}
```

- [ ] **Step 2: Update the `createSession` HTTP helpers**

In each of `content/handler_test.go`, `identity/handler_test.go`, `moderation/handler_test.go`, replace the `createSession(t, serverURL, mailer, email)` helper with a username/password register:

```go
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
```

Update every call site from `createSession(t, serverURL, mailer, "foo@example.com")` to `createSession(t, serverURL, "foo1", "password123")`. Keep each test's usernames unique per test (they already reset tables).

- [ ] **Step 3: Replace `FindOrCreateUserByEmail` in service tests**

In `content/service_test.go`, `identity/service_test.go`, replace:

```go
u, err := authRepo.FindOrCreateUserByEmail(ctx, email)
```

with:

```go
u, err := authRepo.CreateUser(ctx, "tester1", "hash")
```

Adjust usernames per test to avoid collisions (each test clears tables). Where the test needs a specific email for downstream assertions, keep the email references but create the user via `CreateUser` with a distinct username.

- [ ] **Step 4: Update `newTestConfig` in cross-module test files**

Add `StaffUsernames: []string{"admin"}` where tests assert staff claims (moderation). For moderation handler tests that assert `isStaff`, register the user with username `admin` (case-insensitive match).

- [ ] **Step 5: Run all backend tests**

Run:
```bash
cd services/api && TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./... -count=1 -p 1
```
Expected: PASS. Fix any remaining call sites the compiler flags.

- [ ] **Step 6: Run vet and fmt**

Run:
```bash
cd services/api && go vet ./... && go fmt ./...
```
Expected: no output, exit 0.

- [ ] **Step 7: Commit**

```bash
git add services/api/internal/auth/turnstile.go services/api/internal/content services/api/internal/identity services/api/internal/moderation
git commit -m "test(auth): update cross-module tests to username/password"
```

---

## Phase 2: OpenAPI contract

### Task 13: Update the OpenAPI contract

**Files:**
- Modify: `contracts/openapi/openapi.yaml`

- [ ] **Step 1: Replace the auth endpoints**

Replace the `/v1/auth/email-codes` and `/v1/auth/email-sessions` path blocks (lines ~82-135) with:

```yaml
  /v1/auth/register:
    post:
      operationId: register
      summary: Create an account with username and password
      description: |
        Creates a user and returns an access/refresh token pair. Requires a
        Cloudflare Turnstile token.
      security: []
      tags:
        - Auth
      parameters:
        - $ref: "#/components/parameters/IdempotencyKey"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/RegisterRequest"
      responses:
        "201":
          description: Account created and session started.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Session"
        "400":
          $ref: "#/components/responses/BadRequest"
        "409":
          description: Username already taken.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
        "429":
          $ref: "#/components/responses/RateLimited"
  /v1/auth/login:
    post:
      operationId: login
      summary: Sign in with username and password
      description: |
        Verifies the credentials and returns an access/refresh token pair.
        Requires a Cloudflare Turnstile token.
      security: []
      tags:
        - Auth
      parameters:
        - $ref: "#/components/parameters/IdempotencyKey"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/LoginRequest"
      responses:
        "200":
          description: Session created.
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Session"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "429":
          $ref: "#/components/responses/RateLimited"
```

- [ ] **Step 2: Replace the request schemas**

Replace `EmailCodeRequest` and `EmailSessionRequest` (lines ~2425-2454) with:

```yaml
    RegisterRequest:
      type: object
      required:
        - username
        - password
        - turnstileToken
      properties:
        username:
          type: string
          pattern: "^[A-Za-z][A-Za-z0-9_]{2,19}$"
          maxLength: 20
        password:
          type: string
          minLength: 8
          maxLength: 128
        turnstileToken:
          type: string
          description: Cloudflare Turnstile response token.
    LoginRequest:
      type: object
      required:
        - username
        - password
        - turnstileToken
      properties:
        username:
          type: string
          pattern: "^[A-Za-z][A-Za-z0-9_]{2,19}$"
          maxLength: 20
        password:
          type: string
          minLength: 8
          maxLength: 128
        turnstileToken:
          type: string
          description: Cloudflare Turnstile response token.
```

- [ ] **Step 3: Add userId to the Session schema**

In the `Session` schema, add `userId` to `required` and `properties`:

```yaml
      required:
        - accessToken
        - refreshToken
        - tokenType
        - expiresIn
        - userId
        - personaId
      properties:
        accessToken:
          type: string
        refreshToken:
          type: string
        tokenType:
          type: string
          enum:
            - Bearer
        expiresIn:
          type: integer
          description: Access token lifetime in seconds.
        userId:
          type: string
          format: uuid
        personaId:
          type: string
          format: uuid
          nullable: true
          description: Default persona if one has been set.
        isStaff:
          type: boolean
          description: True when the authenticated account has staff privileges.
```

- [ ] **Step 4: Update DELETE /v1/me**

Change `AccountDeletionRequest`:

```yaml
    AccountDeletionRequest:
      type: object
      required:
        - password
      properties:
        password:
          type: string
          minLength: 8
```

Update the description on the `/v1/me` delete path to say "Requires the account password" instead of "verification code".

- [ ] **Step 5: Update the bearer-token description**

Update the `Bearer` description at line ~1537 to reference `POST /v1/auth/login` or `POST /v1/auth/register`.

- [ ] **Step 6: Validate the contract**

Run:
```bash
cd ~/yiguan && make validate-contract
```
Expected: PASS (no output or success message from `scripts/validate-openapi.sh`).

- [ ] **Step 7: Commit**

```bash
git add contracts/openapi/openapi.yaml
git commit -m "feat(api): username/password auth contract with turnstile"
```

---

## Phase 3: Android

### Task 14: Add Turnstile site key to BuildConfig

**Files:**
- Modify: `apps/android/app/build.gradle.kts`
- Modify: `apps/android/gradle/libs.versions.toml`

- [ ] **Step 1: Add the BuildConfig field**

In `defaultConfig`, after the `buildConfigField("String", "API_BASE_URL", ...)` line:

```kotlin
        buildConfigField("String", "TURNSTILE_SITE_KEY", "\"${project.findProperty("turnstileSiteKey") ?: "1x00000000000000000000AA"}\"")
```

- [ ] **Step 2: Commit**

```bash
git add apps/android/app/build.gradle.kts
git commit -m "feat(android): add turnstile site key build config"
```

---

### Task 15: Turnstile WebView composable

**Files:**
- Create: `apps/android/app/src/main/java/app/rebuild/social/core/design/components/TurnstileWebView.kt`

- [ ] **Step 1: Write the composable**

```kotlin
package app.rebuild.social.core.design.components

import android.annotation.SuppressLint
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView

/**
 * Embeds the Cloudflare Turnstile widget in a WebView (Turnstile has no native
 * Android SDK). Emits the token via [onSuccess] or a message via [onError].
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TurnstileWebView(
    siteKey: String,
    onSuccess: (String) -> Unit,
    onError: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    val bridge = remember { TurnstileJsBridge(onSuccess, onError) }

    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.setSupportMultipleWindows(false)
                settings.userAgentString = settings.userAgentString
                addJavascriptInterface(bridge, "LanternTurnstile")
                webViewClient = WebViewClient()
                loadDataWithBaseURL(
                    null,
                    turnstileHtml(siteKey),
                    "text/html",
                    "utf-8",
                    null
                )
            }
        }
    )
}

private class TurnstileJsBridge(
    private val onSuccess: (String) -> Unit,
    private val onError: (String) -> Unit
) {
    @JavascriptInterface
    fun onSuccess(token: String) = onSuccess(token)

    @JavascriptInterface
    fun onError(message: String) = onError(message)
}

private fun turnstileHtml(siteKey: String): String = """
    <!DOCTYPE html>
    <html>
    <head>
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <script src="https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit"></script>
    </head>
    <body style="margin:0;display:flex;justify-content:center;padding:4px;">
      <div id="cf-turnstile"></div>
      <script>
        function renderTurnstile() {
          if (!window.turnstile) { setTimeout(renderTurnstile, 200); return; }
          window.turnstile.render('cf-turnstile', {
            sitekey: '$siteKey',
            callback: function (token) { LanternTurnstile.onSuccess(token); },
            'error-callback': function () { LanternTurnstile.onError('challenge_error'); },
            'expired-callback': function () { LanternTurnstile.onError('challenge_expired'); }
          });
        }
        renderTurnstile();
      </script>
    </body>
    </html>
""".trimIndent()
```

- [ ] **Step 2: Verify it compiles**

Run:
```bash
cd apps/android && ./gradlew :app:compileDebugKotlin
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 3: Commit**

```bash
git add apps/android/app/src/main/java/app/rebuild/social/core/design/components/TurnstileWebView.kt
git commit -m "feat(android): add turnstile webview composable"
```

---

### Task 16: Update the Android network layer

**Files:**
- Modify: `apps/android/app/src/main/java/app/rebuild/social/core/network/ApiClient.kt`
- Modify: `apps/android/app/src/main/java/app/rebuild/social/core/network/LanternApiClient.kt`

- [ ] **Step 1: Replace ApiClient models and interface**

Replace `ApiClient.kt` entirely:

```kotlin
package app.rebuild.social.core.network

import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

interface ApiClient {
    suspend fun register(request: RegisterRequest): Result<AuthSession>
    suspend fun login(request: LoginRequest): Result<AuthSession>
}

@kotlinx.serialization.Serializable
data class RegisterRequest(
    val username: String,
    val password: String,
    val turnstileToken: String
)

@kotlinx.serialization.Serializable
data class LoginRequest(
    val username: String,
    val password: String,
    val turnstileToken: String
)

@kotlinx.serialization.Serializable
data class AuthSession(
    val accessToken: String,
    val refreshToken: String,
    val tokenType: String,
    val expiresIn: Int,
    val userId: String,
    val personaId: String? = null,
    val isStaff: Boolean = false
)

interface LanternApiService {
    @POST("v1/auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<AuthSession>

    @POST("v1/auth/login")
    suspend fun login(@Body request: LoginRequest): Response<AuthSession>
}
```

- [ ] **Step 2: Update LanternApiClient**

Replace `LanternApiClient.kt`:

```kotlin
package app.rebuild.social.core.network

import javax.inject.Inject

class LanternApiClient @Inject constructor(
    private val service: LanternApiService
) : ApiClient {

    override suspend fun register(request: RegisterRequest): Result<AuthSession> {
        return runApiCall { service.register(request) }
    }

    override suspend fun login(request: LoginRequest): Result<AuthSession> {
        return runApiCall { service.login(request) }
    }

    private suspend fun <T : Any> runApiCall(call: suspend () -> retrofit2.Response<T>): Result<T> {
        return try {
            val response = call()
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    Result.success(body)
                } else {
                    Result.failure(
                        ApiException(
                            ApiError.Malformed(
                                message = "The server returned an empty response.",
                                requestId = response.headers()["X-Request-Id"]
                            )
                        )
                    )
                }
            } else {
                Result.failure(ApiException(response.toApiError()))
            }
        } catch (e: Exception) {
            Result.failure(ApiException(e.toApiError()))
        }
    }
}
```

- [ ] **Step 3: Verify compile**

Run:
```bash
cd apps/android && ./gradlew :app:compileDebugKotlin
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 4: Commit**

```bash
git add apps/android/app/src/main/java/app/rebuild/social/core/network/ApiClient.kt apps/android/app/src/main/java/app/rebuild/social/core/network/LanternApiClient.kt
git commit -m "feat(android): username/password api client"
```

---

### Task 17: AuthViewModel

**Files:**
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/AuthViewModel.kt`

- [ ] **Step 1: Write the view model**

```kotlin
package app.rebuild.social.feature.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.rebuild.social.core.network.ApiClient
import app.rebuild.social.core.network.LoginRequest
import app.rebuild.social.core.network.RegisterRequest
import app.rebuild.social.core.session.Session
import app.rebuild.social.core.session.SessionStore
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.temporal.ChronoUnit
import javax.inject.Inject

sealed interface AuthUiState {
    data object Idle : AuthUiState
    data object Loading : AuthUiState
    data class Error(val message: String) : AuthUiState
}

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val apiClient: ApiClient,
    private val sessionStore: SessionStore
) : ViewModel() {

    private val _uiState = MutableStateFlow<AuthUiState>(AuthUiState.Idle)
    val uiState: StateFlow<AuthUiState> = _uiState

    fun register(username: String, password: String, turnstileToken: String, onSuccess: () -> Unit) {
        submit({ apiClient.register(RegisterRequest(username, password, turnstileToken)) }, onSuccess)
    }

    fun login(username: String, password: String, turnstileToken: String, onSuccess: () -> Unit) {
        submit({ apiClient.login(LoginRequest(username, password, turnstileToken)) }, onSuccess)
    }

    private fun submit(block: suspend () -> Result<app.rebuild.social.core.network.AuthSession>, onSuccess: () -> Unit) {
        _uiState.value = AuthUiState.Loading
        viewModelScope.launch {
            block().onSuccess { session ->
                sessionStore.save(
                    Session(
                        accessToken = session.accessToken,
                        refreshToken = session.refreshToken,
                        userId = session.userId,
                        activePersonaId = session.personaId,
                        expiresAt = Instant.now().plus(session.expiresIn.toLong(), ChronoUnit.SECONDS)
                    )
                )
                _uiState.value = AuthUiState.Idle
                onSuccess()
            }.onFailure { e ->
                _uiState.value = AuthUiState.Error(e.message ?: "Authentication failed")
            }
        }
    }
}
```

- [ ] **Step 2: Verify compile**

Run:
```bash
cd apps/android && ./gradlew :app:compileDebugKotlin
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 3: Commit**

```bash
git add apps/android/app/src/main/java/app/rebuild/social/feature/auth/AuthViewModel.kt
git commit -m "feat(android): auth view model for register/login"
```

---

### Task 18: Login and Register screens

**Files:**
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/LoginScreen.kt`
- Create: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/RegisterScreen.kt`
- Delete: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/EmailSignInScreen.kt`
- Delete: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/VerificationScreen.kt`
- Delete: `apps/android/app/src/main/java/app/rebuild/social/feature/auth/VerificationCodeInput.kt`

- [ ] **Step 1: Write LoginScreen**

```kotlin
package app.rebuild.social.feature.auth

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.compose.ui.text.input.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import app.rebuild.social.BuildConfig
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton
import app.rebuild.social.core.design.components.LanternInput
import app.rebuild.social.core.design.components.TurnstileWebView

@Composable
fun LoginScreen(
    onSignInSubmitted: () -> Unit,
    onGoToRegister: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: AuthViewModel = hiltViewModel()
) {
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var turnstileToken by remember { mutableStateOf<String?>(null) }
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        modifier = modifier.testTag("login-screen"),
        topBar = {
            TopAppBar(
                title = { Text("登录", style = LanternType.headingLarge) }
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = LanternSpacing.screenHorizontal)
        ) {
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternInput(
                value = username,
                onValueChange = { username = it },
                label = "用户名",
                placeholder = "请输入用户名"
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternInput(
                value = password,
                onValueChange = { password = it },
                label = "密码",
                placeholder = "请输入密码",
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            TurnstileWebView(
                siteKey = BuildConfig.TURNSTILE_SITE_KEY,
                onSuccess = { turnstileToken = it },
                onError = { turnstileToken = null },
                modifier = Modifier.fillMaxWidth()
            )
            if (uiState is AuthUiState.Error) {
                Spacer(modifier = Modifier.height(LanternSpacing.space3))
                Text(
                    text = (uiState as AuthUiState.Error).message,
                    style = LanternType.bodyMedium,
                    color = MaterialTheme.colorScheme.error
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternButton(
                label = "登录",
                onClick = {
                    val token = turnstileToken
                    if (token != null) {
                        viewModel.login(username, password, token, onSignInSubmitted)
                    }
                },
                variant = ButtonVariant.FilledPrimary,
                modifier = Modifier.fillMaxWidth(),
                enabled = username.isNotBlank() && password.length >= 8 && turnstileToken != null &&
                    uiState != AuthUiState.Loading
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Text(
                text = "没有账号？去注册",
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier
                    .align(Alignment.CenterHorizontally)
                    .clickable { onGoToRegister() }
            )
        }
    }
}
```

- [ ] **Step 2: Write RegisterScreen**

```kotlin
package app.rebuild.social.feature.auth

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.testTag
import androidx.compose.ui.text.input.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.hilt.navigation.compose.hiltViewModel
import app.rebuild.social.BuildConfig
import app.rebuild.social.core.design.LanternSpacing
import app.rebuild.social.core.design.LanternType
import app.rebuild.social.core.design.components.ButtonVariant
import app.rebuild.social.core.design.components.LanternButton
import app.rebuild.social.core.design.components.LanternInput
import app.rebuild.social.core.design.components.TurnstileWebView

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegisterScreen(
    onRegistered: () -> Unit,
    onGoToLogin: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: AuthViewModel = hiltViewModel()
) {
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var confirm by remember { mutableStateOf("") }
    var agreed by remember { mutableStateOf(false) }
    var turnstileToken by remember { mutableStateOf<String?>(null) }
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        modifier = modifier.testTag("register-screen"),
        topBar = {
            TopAppBar(
                title = { Text("注册", style = LanternType.headingLarge) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回"
                        )
                    }
                }
            )
        },
        containerColor = MaterialTheme.colorScheme.background
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = LanternSpacing.screenHorizontal)
        ) {
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternInput(
                value = username,
                onValueChange = { username = it },
                label = "用户名",
                placeholder = "3-20位字母数字下划线"
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternInput(
                value = password,
                onValueChange = { password = it },
                label = "密码",
                placeholder = "至少8位",
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            LanternInput(
                value = confirm,
                onValueChange = { confirm = it },
                label = "确认密码",
                placeholder = "再次输入密码",
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password)
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(
                    checked = agreed,
                    onCheckedChange = { agreed = it }
                )
                Text(
                    text = "我已阅读并同意服务协议与隐私政策",
                    style = LanternType.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            TurnstileWebView(
                siteKey = BuildConfig.TURNSTILE_SITE_KEY,
                onSuccess = { turnstileToken = it },
                onError = { turnstileToken = null },
                modifier = Modifier.fillMaxWidth()
            )
            if (uiState is AuthUiState.Error) {
                Spacer(modifier = Modifier.height(LanternSpacing.space3))
                Text(
                    text = (uiState as AuthUiState.Error).message,
                    style = LanternType.bodyMedium,
                    color = MaterialTheme.colorScheme.error
                )
            }
            Spacer(modifier = Modifier.height(LanternSpacing.space5))
            LanternButton(
                label = "注册",
                onClick = {
                    val token = turnstileToken
                    if (token != null && password == confirm) {
                        viewModel.register(username, password, token, onRegistered)
                    }
                },
                variant = ButtonVariant.FilledPrimary,
                modifier = Modifier.fillMaxWidth(),
                enabled = username.isNotBlank() && password.length >= 8 && password == confirm &&
                    agreed && turnstileToken != null && uiState != AuthUiState.Loading
            )
            Spacer(modifier = Modifier.height(LanternSpacing.space4))
            Text(
                text = "已有账号？去登录",
                style = LanternType.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier
                    .align(Alignment.CenterHorizontally)
                    .clickable { onGoToLogin() }
            )
        }
    }
}
```

- [ ] **Step 3: Delete the old screens**

Delete the three files:

```bash
rm apps/android/app/src/main/java/app/rebuild/social/feature/auth/EmailSignInScreen.kt
rm apps/android/app/src/main/java/app/rebuild/social/feature/auth/VerificationScreen.kt
rm apps/android/app/src/main/java/app/rebuild/social/feature/auth/VerificationCodeInput.kt
```

- [ ] **Step 4: Check the LanternInput signature**

Read `apps/android/app/src/main/java/app/rebuild/social/core/design/components/LanternInput.kt` and confirm the parameter names (`label`, `placeholder`, `visualTransformation`, `keyboardOptions`). Adjust the screen calls above to match the actual signature (e.g., if `visualTransformation` is not a param, drop it or pass a `KeyboardOptions`). Do not guess — read the file first.

- [ ] **Step 5: Verify compile**

Run:
```bash
cd apps/android && ./gradlew :app:compileDebugKotlin
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 6: Commit**

```bash
git add apps/android/app/src/main/java/app/rebuild/social/feature/auth/
git commit -m "feat(android): login and register screens"
```

---

### Task 19: Routes + navigation wiring

**Files:**
- Modify: `apps/android/app/src/main/java/app/rebuild/social/navigation/Routes.kt`
- Modify: `apps/android/app/src/main/java/app/rebuild/social/navigation/RootNavigation.kt`

- [ ] **Step 1: Update Routes**

```kotlin
sealed class Routes(val route: String) {
    data object Welcome : Routes("welcome")
    data object Login : Routes("login")
    data object Register : Routes("register")
    data object Feed : Routes("feed")
    // ... unchanged rest
}
```

Remove `EmailSignIn` and `Verification`.

- [ ] **Step 2: Update RootNavigation**

Replace the imports and the two composable blocks:

```kotlin
import app.rebuild.social.feature.auth.LoginScreen
import app.rebuild.social.feature.auth.RegisterScreen
```

Replace the `Welcome` composable's navigate target and the `EmailSignIn`/`Verification` blocks:

```kotlin
            composable(Routes.Welcome.route) {
                WelcomeScreen(
                    onGetStarted = { navController.navigate(Routes.Login.route) }
                )
            }
            composable(Routes.Login.route) {
                LoginScreen(
                    onSignInSubmitted = {
                        navController.navigate(Routes.Feed.route) {
                            popUpTo(Routes.Welcome.route) { inclusive = true }
                        }
                    },
                    onGoToRegister = { navController.navigate(Routes.Register.route) },
                    onBack = { navController.popBackStack() }
                )
            }
            composable(Routes.Register.route) {
                RegisterScreen(
                    onRegistered = {
                        navController.navigate(Routes.Feed.route) {
                            popUpTo(Routes.Welcome.route) { inclusive = true }
                        }
                    },
                    onGoToLogin = { navController.popBackStack() },
                    onBack = { navController.popBackStack() }
                )
            }
```

- [ ] **Step 3: Verify compile**

Run:
```bash
cd apps/android && ./gradlew :app:compileDebugKotlin
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 4: Commit**

```bash
git add apps/android/app/src/main/java/app/rebuild/social/navigation/
git commit -m "feat(android): wire login/register navigation"
```

---

### Task 20: Update Android tests

**Files:**
- Modify: `apps/android/app/src/androidTest/java/app/rebuild/social/navigation/RootNavigationTest.kt`
- Check: `apps/android/app/src/androidTest/java/app/rebuild/social/core/design/DesignComponentsTest.kt`

- [ ] **Step 1: Update the navigation test**

Replace the `email-signin-screen` reference:

```kotlin
    @Test
    fun unauthenticatedFlow_startsAtWelcomeAndNavigatesToSignIn() {
        composeTestRule.setContent {
            RootNavigation(isAuthenticated = false)
        }

        composeTestRule.onNodeWithTag("welcome-screen").assertIsDisplayed()
        composeTestRule.onNodeWithText("开始").performClick()
        composeTestRule.onNodeWithTag("login-screen").assertIsDisplayed()
    }
```

Also update the "Get started" click to match the WelcomeScreen button text ("开始"). If the button text in `WelcomeScreen.kt` is "开始", use that; otherwise use the literal from the file.

- [ ] **Step 2: Check DesignComponentsTest**

Read `DesignComponentsTest.kt`; if it references any deleted screen or the old welcome button text, update accordingly. Otherwise leave it.

- [ ] **Step 3: Run the tests**

Run:
```bash
cd apps/android && ./gradlew :app:connectedDebugAndroidTest --tests "app.rebuild.social.navigation.RootNavigationTest"
```
Note: requires an emulator. If none is available, compile only:
```bash
cd apps/android && ./gradlew :app:compileDebugAndroidTestKotlin
```

- [ ] **Step 4: Commit**

```bash
git add apps/android/app/src/androidTest/
git commit -m "test(android): adapt navigation tests to login screen"
```

---

## Phase 4: Admin web

### Task 21: Admin API client

**Files:**
- Modify: `apps/admin/src/api/client.ts`
- Modify: `apps/admin/src/types.ts`

- [ ] **Step 1: Replace the auth functions**

Replace `sendEmailCode` and `createEmailSession` with:

```ts
export async function login(username: string, password: string, turnstileToken: string): Promise<Session> {
  const session = await request<Session>('POST', '/auth/login', { username, password, turnstileToken }, `admin-login-${Date.now()}`);
  setToken(session.accessToken);
  localStorage.setItem('lantern_admin_refresh', session.refreshToken);
  return session;
}
```

- [ ] **Step 2: Add userId to Session type**

In `src/types.ts`, add `userId: string;` to the `Session` interface (next to `accessToken`).

- [ ] **Step 3: Commit**

```bash
git add apps/admin/src/api/client.ts apps/admin/src/types.ts
git commit -m "feat(admin): username/password login api"
```

---

### Task 22: Admin Login page with Turnstile

**Files:**
- Modify: `apps/admin/src/pages/Login.tsx`

- [ ] **Step 1: Rewrite Login.tsx**

```tsx
import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { login, clearSession } from '../api/client';

declare global {
  interface Window {
    turnstile?: {
      render: (container: string, opts: { sitekey: string; callback: (token: string) => void; 'error-callback': () => void }) => void;
    };
  }
}

const TURNSTILE_SITE_KEY =
  (import.meta as { env?: Record<string, string> }).env?.VITE_TURNSTILE_SITE_KEY ?? '1x00000000000000000000AA';

export default function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const widgetRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!widgetRef.current || !window.turnstile) return;
    window.turnstile.render(widgetRef.current.id, {
      sitekey: TURNSTILE_SITE_KEY,
      callback: (t: string) => setToken(t),
      'error-callback': () => setToken(''),
    });
  }, []);

  const handleLogin = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!token) {
      setError('Please complete the human verification.');
      return;
    }
    setLoading(true);
    try {
      const session = await login(username, password, token);
      if (!session.isStaff) {
        clearSession();
        setError('This account does not have staff access.');
        return;
      }
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (window.turnstile) return;
    const script = document.createElement('script');
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js';
    script.async = true;
    document.head.appendChild(script);
  }, []);

  return (
    <div className="login-container">
      <div className="login-card">
        <h2>Staff Sign In</h2>
        <p className="hint">Enter your staff username and password.</p>
        {error && <div className="error">{error}</div>}
        <form onSubmit={handleLogin}>
          <label htmlFor="username">Username</label>
          <input
            id="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoFocus
          />
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <div ref={widgetRef} id="cf-turnstile" />
          <button type="submit" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Typecheck**

Run:
```bash
cd apps/admin && npx tsc --noEmit
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add apps/admin/src/pages/Login.tsx
git commit -m "feat(admin): login page with turnstile"
```

---

## Phase 5: Final verification

### Task 23: Full regression

**Files:**
- None (verification only)

- [ ] **Step 1: Run the Go test suite**

Run:
```bash
cd services/api && go vet ./... && go test ./... -count=1 -p 1
```
Expected: PASS.

- [ ] **Step 2: Validate the contract**

Run:
```bash
cd ~/yiguan && make validate-contract
```
Expected: PASS.

- [ ] **Step 3: Build the admin app**

Run:
```bash
cd apps/admin && npm install && npx tsc --noEmit && npm run build
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 4: Compile the Android app**

Run:
```bash
cd apps/android && ./gradlew :app:compileDebugKotlin :app:compileDebugAndroidTestKotlin
```
Expected: BUILD SUCCESSFUL.

- [ ] **Step 5: Commit any remaining changes**

```bash
git status
git add -A
git commit -m "chore: finalize username/password auth migration"
```

---

## Self-Review Notes

- **Spec coverage:** register/login endpoints (Tasks 7-9), Turnstile server-side (Task 4), bcrypt (Tasks 1, 7), staff by username (Tasks 3, 7), password deletion (Tasks 7-8, 13), Android WebView + screens (Tasks 15, 18), admin widget (Task 22), contract updates (Task 13), test rewrites (Tasks 10-12, 20).
- **Placeholder scan:** All steps contain concrete code or exact commands. No TBD/TODO.
- **Type consistency:** `TokenResponse.SessionID` and `TokenResponse.UserID` are added in Task 7 and used in Tasks 7/10. `sessionResponse` helper is added in Task 8 and used by register/login. `StubTurnstile` is created in Task 12 (auth package) and referenced by auth tests in Task 10 — to avoid a forward reference, create `StubTurnstile` in Task 4 (with `turnstile.go`) instead, and reference it in Tasks 10-12.
