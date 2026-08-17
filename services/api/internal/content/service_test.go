package content

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/identity"
	"github.com/yiguan/api/internal/platform/config"
)

func deriveUsername(email string) string {
	local := strings.Split(email, "@")[0]
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	u := strings.TrimLeft(b.String(), "0123456789")
	u = strings.TrimLeft(u, "_")
	if len(u) > 20 {
		u = u[:20]
	}
	if u == "" {
		return "user"
	}
	return u
}

func newContentTestService() *Service {
	cfg := &config.Config{
		JWTSigningKey:   "jwt-secret-key-for-tests-only",
		EmailCodeKey:    "email-code-secret-key-for-tests-only",
		EmailFrom:       "noreply@example.com",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
	idRepo := identity.NewPostgresRepository(testDB)
	repo := NewPostgresRepository(testDB)
	return NewService(cfg, repo, idRepo, nil, auth.NewMemoryLimiter())
}

func createUser(t *testing.T, email string) *auth.User {
	t.Helper()
	ctx := context.Background()
	username := deriveUsername(email)
	u, err := auth.NewPostgresRepository(testDB).CreateUser(ctx, username, "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := testDB.Exec(ctx, "UPDATE users SET email_normalized = $2 WHERE id = $1", u.ID, email); err != nil {
		t.Fatalf("set user email: %v", err)
	}
	return u
}

func createTestPersona(t *testing.T, userID, alias string) string {
	t.Helper()
	ctx := context.Background()
	idRepo := identity.NewPostgresRepository(testDB)
	svc := identity.NewService(&config.Config{EmailCodeKey: "test"}, idRepo, auth.NewPostgresRepository(testDB), nil, auth.NewMemoryLimiter(), nil)
	p, err := svc.CreatePersona(ctx, userID, &identity.PersonaCreateRequest{Alias: alias})
	if err != nil {
		t.Fatalf("create persona: %v", err)
	}
	return p.ID
}

func seedTopic(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	var id string
	const sql = `INSERT INTO topics (name, description, category, status) VALUES ($1, $2, $3, $4) RETURNING id`
	if err := testDB.QueryRow(ctx, sql, "svc-test-topic", "test", "Test", "active").Scan(&id); err != nil {
		t.Fatalf("seed topic: %v", err)
	}
	return id
}

func TestService_CreatePost(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-create-post@example.com")
	personaID := createTestPersona(t, u.ID, "svcposter")
	topicID := seedTopic(t)

	post, err := svc.CreatePost(ctx, u.ID, personaID, &PostCreateRequest{TopicID: topicID, Content: "hello"})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if post.Content != "hello" {
		t.Errorf("expected hello, got %s", post.Content)
	}
	if post.ModerationState != "pendingReview" {
		t.Errorf("expected pendingReview, got %s", post.ModerationState)
	}
}

func TestService_CreatePost_TopicRequired(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-topic-required@example.com")
	personaID := createTestPersona(t, u.ID, "svcposter2")

	_, err := svc.CreatePost(ctx, u.ID, personaID, &PostCreateRequest{TopicID: "", Content: "hello"})
	if !errors.Is(err, ErrPostTopicRequired) {
		t.Fatalf("expected ErrPostTopicRequired, got %v", err)
	}
}

func TestService_UpdatePost_WrongAuthor(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u1 := createUser(t, "svc-author1@example.com")
	p1 := createTestPersona(t, u1.ID, "author1")
	u2 := createUser(t, "svc-author2@example.com")
	p2 := createTestPersona(t, u2.ID, "author2")
	topicID := seedTopic(t)

	post, err := svc.CreatePost(ctx, u1.ID, p1, &PostCreateRequest{TopicID: topicID, Content: "mine"})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	content := "hacked"
	_, err = svc.UpdatePost(ctx, u2.ID, p2, post.ID, &PostUpdateRequest{Content: &content})
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
}

func TestService_DeletePost_SetsDeletedState(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-delete-post@example.com")
	personaID := createTestPersona(t, u.ID, "deleter")
	topicID := seedTopic(t)

	post, err := svc.CreatePost(ctx, u.ID, personaID, &PostCreateRequest{TopicID: topicID, Content: "delete me"})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := svc.DeletePost(ctx, u.ID, personaID, post.ID); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	_, err = svc.GetPost(ctx, post.ID, nil)
	if !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("expected ErrPostNotFound after delete, got %v", err)
	}
}

func TestService_ListPosts_ExcludesDeleted(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-list-posts@example.com")
	personaID := createTestPersona(t, u.ID, "lister")
	topicID := seedTopic(t)

	post, err := svc.CreatePost(ctx, u.ID, personaID, &PostCreateRequest{TopicID: topicID, Content: "visible"})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := svc.DeletePost(ctx, u.ID, personaID, post.ID); err != nil {
		t.Fatalf("delete post: %v", err)
	}

	page, err := svc.ListPosts(ctx, nil, "", 20, nil)
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 posts, got %d", len(page.Data))
	}
}

func TestService_FollowTopic_Duplicate(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-follow@example.com")
	personaID := createTestPersona(t, u.ID, "follower")
	topicID := seedTopic(t)

	if err := svc.FollowTopic(ctx, u.ID, personaID, topicID); err != nil {
		t.Fatalf("follow topic: %v", err)
	}
	if err := svc.FollowTopic(ctx, u.ID, personaID, topicID); !errors.Is(err, ErrTopicAlreadyFollowed) {
		t.Fatalf("expected ErrTopicAlreadyFollowed, got %v", err)
	}
}

func TestService_CreateComment_ParentNotFound(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-comment-parent@example.com")
	personaID := createTestPersona(t, u.ID, "commenter")

	_, err := svc.CreateComment(ctx, u.ID, personaID, "00000000-0000-0000-0000-000000000000", &CommentCreateRequest{Content: "orphan"})
	if !errors.Is(err, ErrCommentParentNotFound) {
		t.Fatalf("expected ErrCommentParentNotFound, got %v", err)
	}
}

func TestService_CreateReaction_InvalidType(t *testing.T) {
	cleanTables(t)
	svc := newContentTestService()
	ctx := context.Background()

	u := createUser(t, "svc-reaction-type@example.com")
	personaID := createTestPersona(t, u.ID, "reactor")
	topicID := seedTopic(t)

	post, err := svc.CreatePost(ctx, u.ID, personaID, &PostCreateRequest{TopicID: topicID, Content: "react"})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	_, err = svc.CreateReaction(ctx, u.ID, personaID, "post", post.ID, &ReactionCreateRequest{Type: "love"})
	if !errors.Is(err, ErrReactionInvalidType) {
		t.Fatalf("expected ErrReactionInvalidType, got %v", err)
	}
}
