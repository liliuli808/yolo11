package content

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/identity"
	"github.com/yiguan/api/internal/platform/config"
)

// PersonaLookup provides read-only access to personas for content validation.
type PersonaLookup interface {
	GetPersonaByIDAndRealProfile(ctx context.Context, id, realProfileID string) (*identity.Persona, error)
	GetPersonaByID(ctx context.Context, id string) (*identity.Persona, error)
}

// BlockProvider provides the list of personas blocked by a viewer.
type BlockProvider interface {
	ListBlockedPersonaIDs(ctx context.Context, blockerPersonaID string) ([]string, error)
}

// Service implements the content domain logic.
type Service struct {
	cfg      *config.Config
	repo     Repository
	personas PersonaLookup
	blocks   BlockProvider
	limiter  auth.RateLimiter
}

// NewService creates a new content Service.
func NewService(cfg *config.Config, repo Repository, personas PersonaLookup, blocks BlockProvider, limiter auth.RateLimiter) *Service {
	return &Service{
		cfg:      cfg,
		repo:     repo,
		personas: personas,
		blocks:   blocks,
		limiter:  limiter,
	}
}

// CursorPage is a generic cursor-paginated result.
type CursorPage[T any] struct {
	Data       []T
	NextCursor *string
	HasMore    bool
	Limit      int
}

// TopicPublic is the public view of a topic.
type TopicPublic struct {
	ID            string
	Name          string
	Description   string
	Category      string
	Status        string
	NoteCount     int
	FollowerCount int
	IsFollowed    bool
	CreatedAt     time.Time
}

// PostPublic is the public view of a post.
type PostPublic struct {
	ID              string
	Persona         *identity.Persona
	Topic           *TopicPublic
	Content         string
	Media           []any
	ReactionCounts  map[string]int
	UserReaction    *string
	IsSaved         bool
	ReplyCount      int
	ModerationState string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CommentPublic is the public view of a comment.
type CommentPublic struct {
	ID              string
	Persona         *identity.Persona
	PostID          string
	Content         string
	ReactionCounts  map[string]int
	UserReaction    *string
	ModerationState string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ReactionPublic is the public view of a reaction.
type ReactionPublic struct {
	ID        string
	Persona   *identity.Persona
	Type      string
	CreatedAt time.Time
}

// PostCreateRequest contains the fields accepted when creating a post.
type PostCreateRequest struct {
	PersonaID *string
	TopicID   string
	Content   string
}

// PostUpdateRequest contains the optional fields accepted when updating a post.
type PostUpdateRequest struct {
	TopicID *string
	Content *string
}

// CommentCreateRequest contains the fields accepted when creating a comment.
type CommentCreateRequest struct {
	PersonaID *string
	Content   string
}

// CommentUpdateRequest contains the fields accepted when updating a comment.
type CommentUpdateRequest struct {
	Content string
}

// ReactionCreateRequest contains the fields accepted when creating a reaction.
type ReactionCreateRequest struct {
	Type string
}

const (
	maxPostContentLen    = 2000
	maxCommentContentLen = 2000
	postRateLimit        = 50
	commentRateLimit     = 100
	postRateWindow       = time.Hour
	commentRateWindow    = time.Hour
)

var (
	postRate    = auth.RateLimit{Count: postRateLimit, Window: postRateWindow}
	commentRate = auth.RateLimit{Count: commentRateLimit, Window: commentRateWindow}
)

func (s *Service) resolvePersona(ctx context.Context, userID, activePersonaID string, override *string) (string, error) {
	personaID := activePersonaID
	if override != nil && *override != "" {
		personaID = *override
	}
	p, err := s.personas.GetPersonaByIDAndRealProfile(ctx, personaID, userID)
	if err != nil {
		if err == identity.ErrPersonaNotFound {
			return "", identity.ErrPersonaNotFound
		}
		return "", fmt.Errorf("resolve persona: %w", err)
	}
	if p.Status != "active" {
		return "", identity.ErrPersonaRestricted
	}
	return p.ID, nil
}

// ListTopics returns curated topics, optionally filtered by search query.
func (s *Service) ListTopics(ctx context.Context, query string, cursor string, limit int, viewerPersonaID *string) (*CursorPage[TopicPublic], error) {
	c, err := ParseCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	topics, next, err := s.repo.ListTopics(ctx, query, c, limit)
	if err != nil {
		return nil, err
	}

	out := make([]TopicPublic, len(topics))
	for i, t := range topics {
		out[i] = s.toTopicPublic(ctx, t, viewerPersonaID)
	}

	var nextCursor *string
	if next != nil {
		cs := next.String()
		nextCursor = &cs
	}
	return &CursorPage[TopicPublic]{
		Data:       out,
		NextCursor: nextCursor,
		HasMore:    next != nil,
		Limit:      limit,
	}, nil
}

// GetTopic returns a single topic by ID.
func (s *Service) GetTopic(ctx context.Context, id string, viewerPersonaID *string) (*TopicPublic, error) {
	t, err := s.repo.GetTopic(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status == "hidden" {
		return nil, ErrTopicHidden
	}
	pub := s.toTopicPublic(ctx, t, viewerPersonaID)
	return &pub, nil
}

func (s *Service) toTopicPublic(ctx context.Context, t *Topic, viewerPersonaID *string) TopicPublic {
	pub := TopicPublic{
		ID:            t.ID,
		Name:          t.Name,
		Description:   t.Description,
		Category:      t.Category,
		Status:        t.Status,
		NoteCount:     t.NoteCount,
		FollowerCount: t.FollowerCount,
		CreatedAt:     t.CreatedAt,
	}
	if viewerPersonaID != nil && *viewerPersonaID != "" {
		followed, _ := s.repo.GetTopicFollow(ctx, *viewerPersonaID, t.ID)
		pub.IsFollowed = followed
	}
	return pub
}

// FollowTopic adds a topic follow for the active persona.
func (s *Service) FollowTopic(ctx context.Context, userID, activePersonaID, topicID string) error {
	if _, err := s.resolvePersona(ctx, userID, activePersonaID, nil); err != nil {
		return err
	}
	if _, err := s.repo.GetTopic(ctx, topicID); err != nil {
		return err
	}
	if err := s.repo.FollowTopic(ctx, activePersonaID, topicID); err != nil {
		return err
	}
	return nil
}

// UnfollowTopic removes a topic follow for the active persona.
func (s *Service) UnfollowTopic(ctx context.Context, userID, activePersonaID, topicID string) error {
	if _, err := s.resolvePersona(ctx, userID, activePersonaID, nil); err != nil {
		return err
	}
	if err := s.repo.UnfollowTopic(ctx, activePersonaID, topicID); err != nil {
		return err
	}
	return nil
}

// CreatePost publishes a new post.
func (s *Service) CreatePost(ctx context.Context, userID, activePersonaID string, req *PostCreateRequest) (*PostPublic, error) {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, req.PersonaID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Content) == "" || len(req.Content) > maxPostContentLen {
		return nil, ErrPostContentDisallowed
	}
	if strings.TrimSpace(req.TopicID) == "" {
		return nil, ErrPostTopicRequired
	}
	if _, err := s.repo.GetTopic(ctx, req.TopicID); err != nil {
		return nil, err
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("post", personaID), postRate); err != nil || !allowed {
		return nil, &auth.RateLimitError{RetryAfter: retryAfter}
	}

	p := &Post{
		PersonaID:       personaID,
		TopicID:         req.TopicID,
		Content:         req.Content,
		ModerationState: "pendingReview",
		ReactionCounts:  map[string]int{},
	}
	if err := s.repo.CreatePost(ctx, p); err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}
	_ = s.repo.IncrementTopicNoteCount(ctx, p.TopicID, 1)

	return s.toPostPublic(ctx, p, &personaID)
}

// GetIdempotencyResponse returns a stored idempotent response, if any.
func (s *Service) GetIdempotencyResponse(ctx context.Context, key, method, path string) (*IdempotencyResponse, error) {
	return s.repo.GetIdempotencyResponse(ctx, key, method, path)
}

// SaveIdempotencyResponse stores an HTTP response for idempotency replay.
func (s *Service) SaveIdempotencyResponse(ctx context.Context, key, method, path string, status int, body []byte) error {
	return s.repo.SaveIdempotencyResponse(ctx, key, method, path, status, body)
}

// GetPost returns a single post.
func (s *Service) GetPost(ctx context.Context, id string, viewerPersonaID *string) (*PostPublic, error) {
	p, err := s.repo.GetPost(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.ModerationState == "deleted" {
		return nil, ErrPostNotFound
	}
	if p.ModerationState != "published" {
		if viewerPersonaID == nil || *viewerPersonaID != p.PersonaID {
			return nil, ErrPostNotFound
		}
	}
	return s.toPostPublic(ctx, p, viewerPersonaID)
}

// UpdatePost edits a post authored by the active persona.
func (s *Service) UpdatePost(ctx context.Context, userID, activePersonaID, postID string, req *PostUpdateRequest) (*PostPublic, error) {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.GetPostByIDAndPersona(ctx, postID, personaID)
	if err != nil {
		return nil, err
	}
	if p.ModerationState != "published" && p.ModerationState != "pendingReview" {
		return nil, ErrPostInvalidState
	}
	if req.Content != nil {
		if strings.TrimSpace(*req.Content) == "" || len(*req.Content) > maxPostContentLen {
			return nil, ErrPostContentDisallowed
		}
		p.Content = *req.Content
	}
	if req.TopicID != nil {
		if strings.TrimSpace(*req.TopicID) == "" {
			return nil, ErrPostTopicRequired
		}
		if _, err := s.repo.GetTopic(ctx, *req.TopicID); err != nil {
			return nil, err
		}
		oldTopicID := p.TopicID
		p.TopicID = *req.TopicID
		_ = s.repo.IncrementTopicNoteCount(ctx, oldTopicID, -1)
		_ = s.repo.IncrementTopicNoteCount(ctx, p.TopicID, 1)
	}
	p.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdatePost(ctx, p); err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}
	return s.toPostPublic(ctx, p, &personaID)
}

// DeletePost soft-deletes a post authored by the active persona.
func (s *Service) DeletePost(ctx context.Context, userID, activePersonaID, postID string) error {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return err
	}
	p, err := s.repo.GetPostByIDAndPersona(ctx, postID, personaID)
	if err != nil {
		return err
	}
	if p.ModerationState == "deleted" {
		return ErrPostInvalidState
	}
	if err := s.repo.DeletePost(ctx, postID); err != nil {
		return err
	}
	_ = s.repo.IncrementTopicNoteCount(ctx, p.TopicID, -1)
	return nil
}

// SavePost saves a post for the active persona.
func (s *Service) SavePost(ctx context.Context, userID, activePersonaID, postID string) error {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return err
	}
	if _, err := s.repo.GetPost(ctx, postID); err != nil {
		return err
	}
	return s.repo.CreateSave(ctx, personaID, postID)
}

// UnsavePost removes a save for the active persona.
func (s *Service) UnsavePost(ctx context.Context, userID, activePersonaID, postID string) error {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return err
	}
	return s.repo.DeleteSave(ctx, personaID, postID)
}

// CreateMediaAsset persists a media asset intent.
func (s *Service) CreateMediaAsset(ctx context.Context, m *MediaAsset) error {
	return s.repo.CreateMediaAsset(ctx, m)
}

// filterBlockedAuthors removes posts whose author is blocked by the viewer.
func (s *Service) filterBlockedAuthors(ctx context.Context, viewerPersonaID *string, posts []*Post) []*Post {
	if viewerPersonaID == nil || *viewerPersonaID == "" || s.blocks == nil {
		return posts
	}
	blocked, err := s.blocks.ListBlockedPersonaIDs(ctx, *viewerPersonaID)
	if err != nil || len(blocked) == 0 {
		return posts
	}
	blockedSet := make(map[string]bool, len(blocked))
	for _, id := range blocked {
		blockedSet[id] = true
	}
	out := make([]*Post, 0, len(posts))
	for _, p := range posts {
		if !blockedSet[p.PersonaID] {
			out = append(out, p)
		}
	}
	return out
}

// ListPosts returns the main feed.
func (s *Service) ListPosts(ctx context.Context, topicID *string, cursor string, limit int, viewerPersonaID *string) (*CursorPage[PostPublic], error) {
	c, err := ParseCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	posts, next, err := s.repo.ListPosts(ctx, topicID, viewerPersonaID, c, limit)
	if err != nil {
		return nil, err
	}
	posts = s.filterBlockedAuthors(ctx, viewerPersonaID, posts)
	return s.toPostPage(ctx, posts, next, limit, viewerPersonaID)
}

// ListTopicPosts returns posts in a topic.
func (s *Service) ListTopicPosts(ctx context.Context, topicID, cursor string, limit int, viewerPersonaID *string) (*CursorPage[PostPublic], error) {
	if _, err := s.repo.GetTopic(ctx, topicID); err != nil {
		return nil, err
	}
	page, err := s.ListPosts(ctx, &topicID, cursor, limit, viewerPersonaID)
	if err != nil {
		return nil, err
	}
	// ListPosts already applies blocked-author filtering.
	return page, nil
}

// ListPersonaPostsForIdentity returns posts authored by a persona in the legacy
// identity CursorPagePost shape used by the public persona endpoint.
func (s *Service) ListPersonaPostsForIdentity(ctx context.Context, personaID, cursor string, limit int, viewerPersonaID *string) (*identity.CursorPagePost, error) {
	page, err := s.ListPersonaPosts(ctx, personaID, cursor, limit, viewerPersonaID)
	if err != nil {
		return nil, err
	}
	data := make([]any, len(page.Data))
	for i, p := range page.Data {
		data[i] = p
	}
	return &identity.CursorPagePost{
		Data:       data,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
		Limit:      page.Limit,
	}, nil
}

// ListPersonaPosts returns posts authored by a persona.
func (s *Service) ListPersonaPosts(ctx context.Context, personaID, cursor string, limit int, viewerPersonaID *string) (*CursorPage[PostPublic], error) {
	p, err := s.personas.GetPersonaByID(ctx, personaID)
	if err != nil {
		if err == identity.ErrPersonaNotFound {
			return nil, identity.ErrPersonaNotFound
		}
		return nil, fmt.Errorf("lookup persona: %w", err)
	}
	if p.Status != "active" {
		return nil, identity.ErrPersonaNotFound
	}
	c, err := ParseCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	posts, next, err := s.repo.ListPersonaPosts(ctx, personaID, viewerPersonaID, c, limit)
	if err != nil {
		return nil, err
	}
	posts = s.filterBlockedAuthors(ctx, viewerPersonaID, posts)
	return s.toPostPage(ctx, posts, next, limit, viewerPersonaID)
}

func (s *Service) toPostPage(ctx context.Context, posts []*Post, next *Cursor, limit int, viewerPersonaID *string) (*CursorPage[PostPublic], error) {
	out := make([]PostPublic, len(posts))
	for i, p := range posts {
		pub, err := s.toPostPublic(ctx, p, viewerPersonaID)
		if err != nil {
			return nil, err
		}
		out[i] = *pub
	}
	var nextCursor *string
	if next != nil {
		cs := next.String()
		nextCursor = &cs
	}
	return &CursorPage[PostPublic]{
		Data:       out,
		NextCursor: nextCursor,
		HasMore:    next != nil,
		Limit:      limit,
	}, nil
}

func (s *Service) toPostPublic(ctx context.Context, p *Post, viewerPersonaID *string) (*PostPublic, error) {
	persona, err := s.personas.GetPersonaByID(ctx, p.PersonaID)
	if err != nil {
		return nil, fmt.Errorf("load post persona: %w", err)
	}
	topic, err := s.repo.GetTopic(ctx, p.TopicID)
	if err != nil {
		return nil, fmt.Errorf("load post topic: %w", err)
	}
	pub := &PostPublic{
		ID:              p.ID,
		Persona:         persona,
		Topic:           s.topicPtrToPublic(topic),
		Content:         p.Content,
		Media:           []any{},
		ReactionCounts:  p.ReactionCounts,
		ReplyCount:      p.ReplyCount,
		ModerationState: p.ModerationState,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
	if viewerPersonaID != nil && *viewerPersonaID != "" {
		if re, err := s.repo.GetReaction(ctx, "post", p.ID, *viewerPersonaID); err == nil {
			pub.UserReaction = &re.Type
		}
		if saved, err := s.repo.GetSave(ctx, *viewerPersonaID, p.ID); err == nil {
			pub.IsSaved = saved
		}
	}
	return pub, nil
}

func (s *Service) topicPtrToPublic(t *Topic) *TopicPublic {
	pub := s.toTopicPublic(context.Background(), t, nil)
	return &pub
}

// CreateComment creates a reply to a post.
func (s *Service) CreateComment(ctx context.Context, userID, activePersonaID, postID string, req *CommentCreateRequest) (*CommentPublic, error) {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, req.PersonaID)
	if err != nil {
		return nil, err
	}
	post, err := s.repo.GetPost(ctx, postID)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			return nil, ErrCommentParentNotFound
		}
		return nil, err
	}
	if post.ModerationState == "deleted" {
		return nil, ErrCommentParentNotFound
	}
	if strings.TrimSpace(req.Content) == "" || len(req.Content) > maxCommentContentLen {
		return nil, ErrCommentContentDisallowed
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("comment", personaID), commentRate); err != nil || !allowed {
		return nil, &auth.RateLimitError{RetryAfter: retryAfter}
	}

	c := &Comment{
		PostID:          postID,
		PersonaID:       personaID,
		Content:         req.Content,
		ModerationState: "pendingReview",
		ReactionCounts:  map[string]int{},
	}
	if err := s.repo.CreateComment(ctx, c); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}
	_ = s.repo.IncrementPostReplyCount(ctx, postID, 1)
	return s.toCommentPublic(ctx, c, &personaID)
}

// GetComment returns a single comment.
func (s *Service) GetComment(ctx context.Context, id string, viewerPersonaID *string) (*CommentPublic, error) {
	c, err := s.repo.GetComment(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.ModerationState == "deleted" {
		return nil, ErrCommentNotFound
	}
	if c.ModerationState != "published" {
		if viewerPersonaID == nil || *viewerPersonaID != c.PersonaID {
			return nil, ErrCommentNotFound
		}
	}
	return s.toCommentPublic(ctx, c, viewerPersonaID)
}

// UpdateComment edits a comment authored by the active persona.
func (s *Service) UpdateComment(ctx context.Context, userID, activePersonaID, commentID string, req *CommentUpdateRequest) (*CommentPublic, error) {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.GetCommentByIDAndPersona(ctx, commentID, personaID)
	if err != nil {
		return nil, err
	}
	if c.ModerationState != "published" && c.ModerationState != "pendingReview" {
		return nil, ErrCommentInvalidState
	}
	if strings.TrimSpace(req.Content) == "" || len(req.Content) > maxCommentContentLen {
		return nil, ErrCommentContentDisallowed
	}
	c.Content = req.Content
	c.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateComment(ctx, c); err != nil {
		return nil, fmt.Errorf("update comment: %w", err)
	}
	return s.toCommentPublic(ctx, c, &personaID)
}

// DeleteComment soft-deletes a comment authored by the active persona.
func (s *Service) DeleteComment(ctx context.Context, userID, activePersonaID, commentID string) error {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return err
	}
	c, err := s.repo.GetCommentByIDAndPersona(ctx, commentID, personaID)
	if err != nil {
		return err
	}
	if c.ModerationState == "deleted" {
		return ErrCommentInvalidState
	}
	if err := s.repo.DeleteComment(ctx, commentID); err != nil {
		return err
	}
	_ = s.repo.IncrementPostReplyCount(ctx, c.PostID, -1)
	return nil
}

// ListComments returns replies to a post.
func (s *Service) ListComments(ctx context.Context, postID, cursor string, limit int, viewerPersonaID *string) (*CursorPage[CommentPublic], error) {
	c, err := ParseCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	comments, next, err := s.repo.ListComments(ctx, postID, viewerPersonaID, c, limit)
	if err != nil {
		return nil, err
	}
	comments = s.filterBlockedAuthorsOnComments(ctx, viewerPersonaID, comments)
	out := make([]CommentPublic, len(comments))
	for i, c := range comments {
		pub, err := s.toCommentPublic(ctx, c, viewerPersonaID)
		if err != nil {
			return nil, err
		}
		out[i] = *pub
	}
	var nextCursor *string
	if next != nil {
		cs := next.String()
		nextCursor = &cs
	}
	return &CursorPage[CommentPublic]{
		Data:       out,
		NextCursor: nextCursor,
		HasMore:    next != nil,
		Limit:      limit,
	}, nil
}

func (s *Service) filterBlockedAuthorsOnComments(ctx context.Context, viewerPersonaID *string, comments []*Comment) []*Comment {
	if viewerPersonaID == nil || *viewerPersonaID == "" || s.blocks == nil {
		return comments
	}
	blocked, err := s.blocks.ListBlockedPersonaIDs(ctx, *viewerPersonaID)
	if err != nil || len(blocked) == 0 {
		return comments
	}
	blockedSet := make(map[string]bool, len(blocked))
	for _, id := range blocked {
		blockedSet[id] = true
	}
	out := make([]*Comment, 0, len(comments))
	for _, c := range comments {
		if !blockedSet[c.PersonaID] {
			out = append(out, c)
		}
	}
	return out
}

func (s *Service) toCommentPublic(ctx context.Context, c *Comment, viewerPersonaID *string) (*CommentPublic, error) {
	persona, err := s.personas.GetPersonaByID(ctx, c.PersonaID)
	if err != nil {
		return nil, fmt.Errorf("load comment persona: %w", err)
	}
	pub := &CommentPublic{
		ID:              c.ID,
		Persona:         persona,
		PostID:          c.PostID,
		Content:         c.Content,
		ReactionCounts:  c.ReactionCounts,
		ModerationState: c.ModerationState,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if viewerPersonaID != nil && *viewerPersonaID != "" {
		if re, err := s.repo.GetReaction(ctx, "comment", c.ID, *viewerPersonaID); err == nil {
			pub.UserReaction = &re.Type
		}
	}
	return pub, nil
}

// CreateReaction adds or replaces the active persona's reaction on a target.
func (s *Service) CreateReaction(ctx context.Context, userID, activePersonaID, targetType, targetID string, req *ReactionCreateRequest) (*ReactionSummary, error) {
	if req.Type != "like" {
		return nil, ErrReactionInvalidType
	}
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return nil, err
	}
	if err := s.ensureReactionTargetExists(ctx, targetType, targetID); err != nil {
		return nil, err
	}

	re := &Reaction{
		TargetType: targetType,
		TargetID:   targetID,
		PersonaID:  personaID,
		Type:       req.Type,
	}
	if err := s.repo.UpsertReaction(ctx, re); err != nil {
		return nil, fmt.Errorf("upsert reaction: %w", err)
	}
	if err := s.repo.RecomputeReactionCounts(ctx, targetType, targetID); err != nil {
		return nil, err
	}
	return s.ReactionSummary(ctx, targetType, targetID, personaID)
}

// DeleteReaction removes the active persona's reaction from a target.
func (s *Service) DeleteReaction(ctx context.Context, userID, activePersonaID, targetType, targetID string) error {
	personaID, err := s.resolvePersona(ctx, userID, activePersonaID, nil)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteReaction(ctx, targetType, targetID, personaID); err != nil {
		return err
	}
	return s.repo.RecomputeReactionCounts(ctx, targetType, targetID)
}

// ReactionSummary returns reaction counts and the viewer's own reaction.
type ReactionSummary struct {
	ReactionCounts map[string]int
	UserReaction   *string
}

// ReactionSummary returns the reaction summary for a target.
func (s *Service) ReactionSummary(ctx context.Context, targetType, targetID, viewerPersonaID string) (*ReactionSummary, error) {
	var counts map[string]int
	switch targetType {
	case "post":
		p, err := s.repo.GetPost(ctx, targetID)
		if err != nil {
			return nil, err
		}
		counts = p.ReactionCounts
	case "comment":
		c, err := s.repo.GetComment(ctx, targetID)
		if err != nil {
			return nil, err
		}
		counts = c.ReactionCounts
	default:
		return nil, ErrReactionInvalidType
	}

	summary := &ReactionSummary{ReactionCounts: counts}
	if viewerPersonaID != "" {
		if re, err := s.repo.GetReaction(ctx, targetType, targetID, viewerPersonaID); err == nil {
			summary.UserReaction = &re.Type
		}
	}
	return summary, nil
}

// ListReactions returns reactions on a target.
func (s *Service) ListReactions(ctx context.Context, targetType, targetID, cursor string, limit int) (*CursorPage[ReactionPublic], error) {
	c, err := ParseCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	reactions, next, err := s.repo.ListReactions(ctx, targetType, targetID, c, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ReactionPublic, len(reactions))
	for i, re := range reactions {
		persona, err := s.personas.GetPersonaByID(ctx, re.PersonaID)
		if err != nil {
			return nil, fmt.Errorf("load reaction persona: %w", err)
		}
		out[i] = ReactionPublic{
			ID:        re.ID,
			Persona:   persona,
			Type:      re.Type,
			CreatedAt: re.CreatedAt,
		}
	}
	var nextCursor *string
	if next != nil {
		cs := next.String()
		nextCursor = &cs
	}
	return &CursorPage[ReactionPublic]{
		Data:       out,
		NextCursor: nextCursor,
		HasMore:    next != nil,
		Limit:      limit,
	}, nil
}

func (s *Service) ensureReactionTargetExists(ctx context.Context, targetType, targetID string) error {
	switch targetType {
	case "post":
		p, err := s.repo.GetPost(ctx, targetID)
		if err != nil {
			return ErrReactionTargetNotFound
		}
		if p.ModerationState == "deleted" {
			return ErrReactionTargetNotFound
		}
	case "comment":
		c, err := s.repo.GetComment(ctx, targetID)
		if err != nil {
			return ErrReactionTargetNotFound
		}
		if c.ModerationState == "deleted" {
			return ErrReactionTargetNotFound
		}
	default:
		return ErrReactionInvalidType
	}
	return nil
}

func rateLimitKey(prefix, personaID string) string {
	return fmt.Sprintf("content:%s:persona:%s", prefix, personaID)
}
