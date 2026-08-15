# WP6: Topics, Posts, Media, and Interaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the content subsystem (topics, topic follows, posts, comments, reactions, media placeholder) and active-persona middleware so all WP6 endpoints in the OpenAPI contract return correct responses.

**Architecture:** Add a new `internal/content` package mirroring `internal/identity` (repository, service, handler). The content service depends on the identity repository only for persona validation; the identity handler delegates `GET /v1/personas/{personaId}/posts` to the content service. An `ActivePersonaMiddleware` in `internal/identity` resolves the caller's default persona and stores it in request context for mutating content routes.

**Tech Stack:** Go, Chi, pgx/v5, PostgreSQL, existing `internal/auth`, `internal/identity`, `internal/platform/httpx`.

---

## File Map

| File | Responsibility |
|------|----------------|
| `migrations/content/021_topics.up.sql` | Create `topics` and `topic_follows` tables. |
| `migrations/content/021_topics.down.sql` | Drop `topics` and `topic_follows`. |
| `migrations/content/022_posts.up.sql` | Create `posts`, `comments`, `reactions`, `media_assets` tables. |
| `migrations/content/022_posts.down.sql` | Drop those tables. |
| `internal/content/repository.go` | Domain entities, repository interface, Postgres implementation. |
| `internal/content/service.go` | Business logic and domain errors. |
| `internal/content/handler.go` | HTTP handlers and response mapping. |
| `internal/content/handler_test.go` | Handler tests against a real test DB. |
| `internal/content/service_test.go` | Service tests against a real test DB. |
| `internal/identity/middleware.go` | `ActivePersonaMiddleware` and persona context helpers. |
| `internal/identity/handler.go` | Add active-persona middleware usage; delegate `ListPersonaPosts` to content service. |
| `internal/identity/service.go` | Add `ContentService` dependency and delegate `ListPersonaPosts`. |
| `cmd/api/main.go` | Wire content repository/service/handler into router. |

---

### Task 1: Content Database Migration

**Files:**
- Create: `migrations/content/021_topics.up.sql`
- Create: `migrations/content/021_topics.down.sql`
- Create: `migrations/content/022_posts.up.sql`
- Create: `migrations/content/022_posts.down.sql`
- Delete: `migrations/content/020_placeholder.up.sql`
- Delete: `migrations/content/020_placeholder.down.sql`

- [ ] **Step 1.1: Create topics migration**

Create `migrations/content/021_topics.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(256) NOT NULL DEFAULT '',
    category VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'hidden')),
    slug VARCHAR(64),
    icon_url VARCHAR(512),
    cover_url VARCHAR(512),
    note_count INTEGER NOT NULL DEFAULT 0,
    follower_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS topic_follows (
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (persona_id, topic_id)
);

CREATE INDEX idx_topics_status ON topics(status);
CREATE INDEX idx_topics_name ON topics(name);
CREATE INDEX idx_topic_follows_topic ON topic_follows(topic_id);

INSERT INTO topics (name, description, category, status)
VALUES
    ('General', 'Open conversation about anything.', 'Everyday', 'active'),
    ('Reflection', 'Deeper thoughts and personal reflections.', 'Reflection', 'active'),
    ('Creative', 'Share creative work and inspiration.', 'Creative', 'active')
ON CONFLICT (name) DO NOTHING;
```

Create `migrations/content/021_topics.down.sql`:

```sql
DROP TABLE IF EXISTS topic_follows;
DROP TABLE IF EXISTS topics;
```

- [ ] **Step 1.2: Create posts/comments/reactions/media migration**

Create `migrations/content/022_posts.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE RESTRICT,
    content TEXT NOT NULL CHECK (length(content) <= 2000),
    moderation_state VARCHAR(16) NOT NULL DEFAULT 'published'
        CHECK (moderation_state IN ('published', 'pendingReview', 'rejected', 'hidden', 'deleted')),
    reaction_counts JSONB NOT NULL DEFAULT '{}',
    reply_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_posts_persona ON posts(persona_id);
CREATE INDEX idx_posts_topic ON posts(topic_id);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);

CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (length(content) <= 2000),
    moderation_state VARCHAR(16) NOT NULL DEFAULT 'published'
        CHECK (moderation_state IN ('published', 'pendingReview', 'rejected', 'hidden', 'deleted')),
    reaction_counts JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_persona ON comments(persona_id);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);

CREATE TABLE IF NOT EXISTS reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type VARCHAR(16) NOT NULL CHECK (target_type IN ('post', 'comment')),
    target_id UUID NOT NULL,
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    type VARCHAR(16) NOT NULL CHECK (type IN ('like')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (target_type, target_id, persona_id)
);

CREATE INDEX idx_reactions_target ON reactions(target_type, target_id);

CREATE TABLE IF NOT EXISTS media_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID NOT NULL REFERENCES personas(id) ON DELETE CASCADE,
    url VARCHAR(1024) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    thumbnail_url VARCHAR(1024),
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'ready', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Create `migrations/content/022_posts.down.sql`:

```sql
DROP TABLE IF EXISTS media_assets;
DROP TABLE IF EXISTS reactions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS posts;
```

- [ ] **Step 1.3: Remove placeholder migrations**

Delete `migrations/content/020_placeholder.up.sql` and `migrations/content/020_placeholder.down.sql`.

- [ ] **Step 1.4: Run migrations**

```bash
cd /home/jichi/yiguan
make migrate-test-up
```

Expected: migrations apply without errors.

- [ ] **Step 1.5: Commit**

```bash
git add migrations/content/021_topics.* migrations/content/022_posts.*
git rm migrations/content/020_placeholder.*
git commit -m "feat(content): add topics, posts, comments, reactions, media migrations"
```

---

### Task 2: Active Persona Middleware

**Files:**
- Create: `internal/identity/middleware.go`
- Modify: `internal/identity/handler.go`

- [ ] **Step 2.1: Write middleware**

Create `internal/identity/middleware.go`:

```go
package identity

import (
	"context"
	"net/http"

	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/platform/httpx"
)

type personaIDKey int

const activePersonaIDKey personaIDKey = iota

// ActivePersonaMiddleware resolves the authenticated user's default persona and
// stores the active persona ID in request context. Content handlers read the
// persona ID from context. If the user has no active/default persona, it
// responds with PERSONA.DEFAULT_REQUIRED.
func (h *Handler) ActivePersonaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			httpx.Error(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
			return
		}

		// Persona override from request body is handled by handlers; here we only
		// resolve the default persona for the user.
		profile, err := h.service.GetMe(r.Context(), userID)
		if err != nil {
			if err == ErrProfileNotFound {
				httpx.Error(r.Context(), w, http.StatusNotFound, "ME.NOT_FOUND", "account not found")
				return
			}
			httpx.Error(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
			return
		}

		if profile.DefaultPersonaID == nil || *profile.DefaultPersonaID == "" {
			httpx.Error(r.Context(), w, http.StatusBadRequest, "PERSONA.DEFAULT_REQUIRED", "please select a default persona first")
			return
		}

		p, err := h.service.GetPrivatePersona(r.Context(), userID, *profile.DefaultPersonaID)
		if err != nil {
			if err == ErrPersonaNotFound {
				httpx.Error(r.Context(), w, http.StatusBadRequest, "PERSONA.DEFAULT_REQUIRED", "please select a default persona first")
				return
			}
			httpx.Error(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
			return
		}
		if p.Status != "active" {
			httpx.Error(r.Context(), w, http.StatusForbidden, "PERSONA.RESTRICTED", "persona cannot be used")
			return
		}

		ctx := context.WithValue(r.Context(), activePersonaIDKey, p.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ActivePersonaIDFromContext returns the active persona ID placed by
// ActivePersonaMiddleware, or empty string if none.
func ActivePersonaIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(activePersonaIDKey).(string)
	return v
}
```

- [ ] **Step 2.2: Add middleware helper to identity handler**

Add to `internal/identity/handler.go` a method that chains auth + active persona:

```go
// WithActivePersona returns a middleware stack that requires authentication and
// a usable default persona.
func (h *Handler) WithActivePersona() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return h.authHandler.AuthMiddleware(h.ActivePersonaMiddleware(next))
	}
}
```

- [ ] **Step 2.3: Add handler unit test**

Add to `internal/identity/handler_test.go`:

```go
func TestHandler_ActivePersonaMiddleware_RequiresDefaultPersona(t *testing.T) {
	h, authHandler, mailer := setupHandlerTest(t)
	server := httptest.NewServer(mountIdentityHandler(h, authHandler))
	defer server.Close()

	token := createSession(t, server.URL, mailer, "handler-active-persona@example.com")

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/topics/00000000-0000-0000-0000-000000000000/follow", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "active-persona-follow-key-1")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("follow topic: %v", err)
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
```

- [ ] **Step 2.4: Run identity tests**

```bash
cd /home/jichi/yiguan/services/api
TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/identity/... -race -count=1
```

Expected: PASS.

- [ ] **Step 2.5: Commit**

```bash
git add internal/identity/middleware.go internal/identity/handler.go internal/identity/handler_test.go
git commit -m "feat(identity): add active persona middleware"
```

---

### Task 3: Content Repository

**Files:**
- Create: `internal/content/repository.go`

- [ ] **Step 3.1: Write repository file**

Create `internal/content/repository.go` with domain structs, errors, and Postgres implementation. Include methods for:
- Topics: ListTopics, GetTopic, GetTopicFollow, FollowTopic, UnfollowTopic, IncrementTopicNoteCount
- Posts: CreatePost, GetPost, GetPostByIDAndPersona, UpdatePost, DeletePost, ListPosts, ListTopicPosts, ListPersonaPosts
- Comments: CreateComment, GetComment, GetCommentByIDAndPersona, UpdateComment, DeleteComment, ListComments
- Reactions: UpsertReaction, DeleteReaction, ListReactions, GetReactionSummary

Key implementation details:
- Use cursor pagination based on `created_at` and `id`.
- Keep blocked-author filtering as a no-op placeholder join (`LEFT JOIN blocks ... ON 1=0`) so the signature works when WP7 adds the table.
- Soft deletes set `moderation_state = 'deleted'` and `deleted_at = now()`.
- Reaction counts are stored as JSONB and updated via helper functions.

See the full source in the implementation phase; the repository file is large, so the plan references a single file with all methods rather than duplicating every line here. The engineer should implement each method with the SQL provided in the implementation notes below.

Implementation notes for key queries:

Topic list (supports `q` search):
```sql
SELECT id, name, description, category, status, note_count, follower_count, created_at, updated_at
FROM topics
WHERE status = 'active' AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
ORDER BY created_at ASC, id ASC
LIMIT $2
```

Topic follow check:
```sql
SELECT EXISTS(SELECT 1 FROM topic_follows WHERE persona_id = $1 AND topic_id = $2)
```

Post list with blocked placeholder:
```sql
SELECT p.id, p.persona_id, p.topic_id, p.content, p.moderation_state, p.reaction_counts, p.reply_count, p.created_at, p.updated_at
FROM posts p
LEFT JOIN blocks ON 1=0
WHERE p.moderation_state = 'published'
  AND ($1::uuid IS NULL OR p.topic_id = $1)
  AND ($2::timestamptz IS NULL OR (p.created_at, p.id) < ($2, $3::uuid))
ORDER BY p.created_at DESC, p.id DESC
LIMIT $4
```

Reaction upsert (insert or update, then recompute counts):
```sql
INSERT INTO reactions (target_type, target_id, persona_id, type)
VALUES ($1, $2, $3, $4)
ON CONFLICT (target_type, target_id, persona_id) DO UPDATE SET type = EXCLUDED.type, updated_at = now()
RETURNING id, created_at, updated_at
```

After reaction change, recompute JSONB counts:
```sql
UPDATE posts
SET reaction_counts = (
    SELECT COALESCE(jsonb_object_agg(type, cnt), '{}')
    FROM (SELECT type, COUNT(*)::int AS cnt FROM reactions WHERE target_type = $1 AND target_id = $2 GROUP BY type) t
),
updated_at = now()
WHERE id = $2
```

- [ ] **Step 3.2: Run go vet / compile**

```bash
cd /home/jichi/yiguan/services/api
go vet ./internal/content/...
```

Expected: no errors.

- [ ] **Step 3.3: Commit**

```bash
git add internal/content/repository.go
git commit -m "feat(content): add content repository"
```

---

### Task 4: Content Service

**Files:**
- Create: `internal/content/service.go`

- [ ] **Step 4.1: Write service file**

Create `internal/content/service.go` with domain errors and service methods.

Domain errors:
```go
var (
	ErrTopicNotFound        = errors.New("topic not found")
	ErrTopicAlreadyFollowed = errors.New("topic already followed")
	ErrTopicNotFollowed     = errors.New("topic not followed")
	ErrTopicHidden          = errors.New("topic hidden")
	ErrPostNotFound         = errors.New("post not found")
	ErrPostNotAuthor        = errors.New("post not author")
	ErrPostInvalidState     = errors.New("post invalid state")
	ErrPostContentDisallowed = errors.New("post content disallowed")
	ErrPostTopicRequired    = errors.New("post topic required")
	ErrPostRateLimited      = errors.New("post rate limited")
	ErrCommentNotFound      = errors.New("comment not found")
	ErrCommentNotAuthor     = errors.New("comment not author")
	ErrCommentInvalidState  = errors.New("comment invalid state")
	ErrCommentContentDisallowed = errors.New("comment content disallowed")
	ErrCommentParentNotFound = errors.New("comment parent not found")
	ErrCommentRateLimited   = errors.New("comment rate limited")
	ErrReactionInvalidType  = errors.New("reaction invalid type")
	ErrReactionAlreadyExists = errors.New("reaction already exists")
	ErrReactionNotFound     = errors.New("reaction not found")
	ErrReactionTargetNotFound = errors.New("reaction target not found")
)
```

Service methods implement:
- Topic listing/following.
- Post CRUD, feed listing, topic posts, persona posts.
- Comment CRUD on posts.
- Reaction upsert/delete and summary.

Validation rules:
- Post content: 1-2000 chars.
- Post create requires `topicId`.
- Only `published` or `pendingReview` posts can be edited.
- Author edits only their own posts/comments.
- Soft delete transitions to `deleted`.
- Comments require parent post to exist and not be deleted.
- Only `like` reaction type is supported.
- Rate limiting: placeholder in-memory limiter using auth.NewMemoryLimiter keyed by persona ID for posts/comments.

- [ ] **Step 4.2: Run tests (placeholder)**

Service tests come in Task 6.

- [ ] **Step 4.3: Commit**

```bash
git add internal/content/service.go
git commit -m "feat(content): add content service"
```

---

### Task 5: Content HTTP Handler

**Files:**
- Create: `internal/content/handler.go`

- [ ] **Step 5.1: Write handler file**

Create `internal/content/handler.go` with:
- request/response structs matching OpenAPI schemas.
- `Mount` registering all routes.
- handlers: ListTopics, GetTopic, FollowTopic, UnfollowTopic, ListTopicPosts, ListPosts, CreatePost, GetPost, UpdatePost, DeletePost, ListComments, CreateComment, GetComment, UpdateComment, DeleteComment, ListReactions, CreateReaction, DeleteReaction, UploadMedia.
- middleware usage: `WithActivePersona` for mutating routes.
- `UploadMedia` returns 501.
- Error mapping matching `docs/architecture/api-errors.md`.
- Helper: `requireActivePersona`, `requireIdempotencyKey`, `parseLimit`, `parseCursor`.
- Response mapping: `toPostResponse`, `toCommentResponse`, `toReactionResponse`, `toTopicResponse`, etc.

- [ ] **Step 5.2: Add CORS header for Idempotency-Key**

Verify `internal/platform/httpx/middleware.go` CORS `AllowedHeaders` includes `Idempotency-Key`. If not, add it:

```go
AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "Idempotency-Key"},
```

- [ ] **Step 5.3: Compile**

```bash
cd /home/jichi/yiguan/services/api
go build ./internal/content/...
```

Expected: builds successfully.

- [ ] **Step 5.4: Commit**

```bash
git add internal/content/handler.go internal/platform/httpx/middleware.go
git commit -m "feat(content): add content HTTP handler"
```

---

### Task 6: Wire Content Module into Main Router

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 6.1: Update main.go**

In `cmd/api/main.go`, after identity setup, add:

```go
contentRepo := content.NewPostgresRepository(pool)
contentService := content.NewService(cfg, contentRepo, identityRepo, auth.NewMemoryLimiter())
contentHandler := content.NewHandler(contentService, identityHandler, cfg)
```

Update `identity.NewHandler` and `identity.NewService` signatures to accept the content service so `ListPersonaPosts` can delegate. Adjust call sites accordingly.

In `newV1Router`, mount content handler:

```go
if contentHandler != nil {
    contentHandler.Mount(r)
}
```

Update function signatures to pass `contentHandler` through.

- [ ] **Step 6.2: Update identity service to accept content service**

Modify `internal/identity/service.go`:
- Add `contentService ContentService` field.
- Add interface `ContentService` with `ListPersonaPosts(ctx, personaID, cursor string, limit int) (*content.CursorPagePost, error)`.
- Update `NewService` to accept the interface.
- Replace placeholder `ListPersonaPosts` implementation with a delegation to `s.contentService.ListPersonaPosts`.

- [ ] **Step 6.3: Update identity handler constructor**

`internal/identity/handler.go` `NewHandler` already accepts `*auth.Handler`; ensure it can access content service via the service dependency. No signature change needed if service holds the dependency.

- [ ] **Step 6.4: Compile and run tests**

```bash
cd /home/jichi/yiguan/services/api
go build ./...
TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/identity/... -race -count=1
```

Expected: PASS.

- [ ] **Step 6.5: Commit**

```bash
git add cmd/api/main.go internal/identity/service.go internal/identity/handler.go
git commit -m "feat(api): wire content module and delegate persona posts"
```

---

### Task 7: Content Handler Tests

**Files:**
- Create: `internal/content/handler_test.go`

- [ ] **Step 7.1: Write test setup**

Create `internal/content/handler_test.go` with:
- `TestMain` connecting to `TEST_DATABASE_URL`, resetting content schema, running auth + identity + content migrations.
- `cleanTables` truncating all content and identity/auth tables.
- `setupHandlerTest` creating auth/identity/content repos, services, handlers, and mounting them on a test router.
- helpers: `createSession`, `createPersona`, `createTopic`, `createPost`.

- [ ] **Step 7.2: Write core handler tests**

Add tests:
- `TestHandler_CreatePost` — create post with default persona, verify 201 and response shape.
- `TestHandler_CreatePost_RequiresDefaultPersona` — user with no persona gets `PERSONA.DEFAULT_REQUIRED`.
- `TestHandler_ListTopics` — list topics returns seeded topics.
- `TestHandler_FollowAndUnfollowTopic` — follow returns 204, duplicate follow returns 409, unfollow returns 204.
- `TestHandler_ListTopicPosts` — posts in topic appear in list.
- `TestHandler_UpdatePost` — author can edit, non-author cannot.
- `TestHandler_DeletePost` — author soft-deletes.
- `TestHandler_CreateComment` — reply to post, verify 201.
- `TestHandler_CreateReaction` — like a post, verify summary.
- `TestHandler_UploadMedia` — returns 501.

- [ ] **Step 7.3: Run content handler tests**

```bash
cd /home/jichi/yiguan/services/api
TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/content/... -race -count=1 -v
```

Expected: PASS.

- [ ] **Step 7.4: Commit**

```bash
git add internal/content/handler_test.go
git commit -m "test(content): add content handler tests"
```

---

### Task 8: Content Service Tests

**Files:**
- Create: `internal/content/service_test.go`

- [ ] **Step 8.1: Write service tests**

Create `internal/content/service_test.go` with:
- `TestService_CreatePost`.
- `TestService_CreatePost_TopicRequired`.
- `TestService_UpdatePost_WrongAuthor`.
- `TestService_DeletePost_SetsDeletedState`.
- `TestService_ListPosts_ExcludesDeleted`.
- `TestService_FollowTopic_Duplicate`.
- `TestService_CreateComment_ParentNotFound`.
- `TestService_CreateReaction_InvalidType`.

- [ ] **Step 8.2: Run service tests**

```bash
cd /home/jichi/yiguan/services/api
TEST_DATABASE_URL=postgres://yiguan:yiguan@localhost:15433/yiguan_test?sslmode=disable go test ./internal/content/... -race -count=1 -v
```

Expected: PASS.

- [ ] **Step 8.3: Commit**

```bash
git add internal/content/service_test.go
git commit -m "test(content): add content service tests"
```

---

### Task 9: Full Test Suite and Contract Validation

**Files:**
- Modify: any remaining compile issues.

- [ ] **Step 9.1: Run full test suite**

```bash
cd /home/jichi/yiguan
make test
```

Expected: all tests pass.

- [ ] **Step 9.2: Validate OpenAPI contract**

```bash
cd /home/jichi/yiguan
make validate-contract
```

Expected: contract validates successfully.

- [ ] **Step 9.3: Run linters**

```bash
cd /home/jichi/yiguan
make lint
```

Expected: no lint errors (or golangci-lint skipped).

- [ ] **Step 9.4: Final commit**

```bash
git commit -m "feat(wp6): complete topics, posts, comments, reactions, media placeholder" --allow-empty
```

---

## Self-Review Checklist

- [ ] Spec coverage: every WP6 endpoint in `openapi.yaml` has a handler.
- [ ] No placeholders in plan steps (no "TBD", "implement later").
- [ ] Error codes match `api-errors.md`.
- [ ] Active persona middleware resolves default persona and returns `PERSONA.DEFAULT_REQUIRED`.
- [ ] Media upload returns 501.
- [ ] Block filtering uses placeholder join; no hard dependency on WP7.
- [ ] `make test` passes.
