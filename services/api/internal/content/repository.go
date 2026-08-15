package content

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Domain errors used by the content subsystem.
var (
	ErrTopicNotFound            = errors.New("topic not found")
	ErrTopicAlreadyFollowed     = errors.New("topic already followed")
	ErrTopicNotFollowed         = errors.New("topic not followed")
	ErrTopicHidden              = errors.New("topic hidden")
	ErrPostNotFound             = errors.New("post not found")
	ErrPostNotAuthor            = errors.New("post not author")
	ErrPostInvalidState         = errors.New("post invalid state")
	ErrPostContentDisallowed    = errors.New("post content disallowed")
	ErrPostTopicRequired        = errors.New("post topic required")
	ErrPostRateLimited          = errors.New("post rate limited")
	ErrCommentNotFound          = errors.New("comment not found")
	ErrCommentNotAuthor         = errors.New("comment not author")
	ErrCommentInvalidState      = errors.New("comment invalid state")
	ErrCommentContentDisallowed = errors.New("comment content disallowed")
	ErrCommentParentNotFound    = errors.New("comment parent not found")
	ErrCommentRateLimited       = errors.New("comment rate limited")
	ErrReactionInvalidType      = errors.New("reaction invalid type")
	ErrReactionAlreadyExists    = errors.New("reaction already exists")
	ErrReactionNotFound         = errors.New("reaction not found")
	ErrReactionTargetNotFound   = errors.New("reaction target not found")
	ErrSaveAlreadyExists        = errors.New("save already exists")
	ErrSaveNotFound             = errors.New("save not found")
)

// Topic is a curated channel.
type Topic struct {
	ID            string
	Name          string
	Description   string
	Category      string
	Status        string
	Slug          *string
	IconURL       *string
	CoverURL      *string
	NoteCount     int
	FollowerCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Post is a note authored by a persona.
type Post struct {
	ID              string
	PersonaID       string
	TopicID         string
	Content         string
	ModerationState string
	ReactionCounts  map[string]int
	ReplyCount      int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// Comment is a reply to a post.
type Comment struct {
	ID              string
	PostID          string
	PersonaID       string
	Content         string
	ModerationState string
	ReactionCounts  map[string]int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

// Reaction is a lightweight engagement on a post or comment.
type Reaction struct {
	ID         string
	TargetType string
	TargetID   string
	PersonaID  string
	Type       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// MediaAsset is a placeholder media metadata record.
type MediaAsset struct {
	ID           string
	PersonaID    string
	URL          string
	MimeType     string
	Width        int
	Height       int
	FileSize     *int64
	Checksum     *string
	ThumbnailURL string
	Status       string
	CreatedAt    time.Time
}

// IdempotencyResponse is a stored HTTP response for idempotency replay.
type IdempotencyResponse struct {
	Status int
	Body   []byte
}

// Cursor is an opaque pagination token based on created_at and id.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// String returns a base64-encoded cursor.
func (c *Cursor) String() string {
	if c == nil {
		return ""
	}
	raw := c.CreatedAt.Format(time.RFC3339Nano) + "|" + c.ID
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// ParseCursor decodes a cursor string.
func ParseCursor(s string) (*Cursor, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, err
	}
	return &Cursor{CreatedAt: ts, ID: parts[1]}, nil
}

// Repository provides persistence for content entities.
type Repository interface {
	// Topics
	ListTopics(ctx context.Context, query string, cursor *Cursor, limit int) ([]*Topic, *Cursor, error)
	GetTopic(ctx context.Context, id string) (*Topic, error)
	GetTopicFollow(ctx context.Context, personaID, topicID string) (bool, error)
	FollowTopic(ctx context.Context, personaID, topicID string) error
	UnfollowTopic(ctx context.Context, personaID, topicID string) error
	IncrementTopicNoteCount(ctx context.Context, topicID string, delta int) error

	// Posts
	CreatePost(ctx context.Context, p *Post) error
	GetPost(ctx context.Context, id string) (*Post, error)
	GetPostByIDAndPersona(ctx context.Context, id, personaID string) (*Post, error)
	UpdatePost(ctx context.Context, p *Post) error
	DeletePost(ctx context.Context, id string) error
	ListPosts(ctx context.Context, topicID *string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error)
	ListTopicPosts(ctx context.Context, topicID string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error)
	ListPersonaPosts(ctx context.Context, personaID string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error)
	IncrementPostReplyCount(ctx context.Context, postID string, delta int) error

	// Comments
	CreateComment(ctx context.Context, c *Comment) error
	GetComment(ctx context.Context, id string) (*Comment, error)
	GetCommentByIDAndPersona(ctx context.Context, id, personaID string) (*Comment, error)
	UpdateComment(ctx context.Context, c *Comment) error
	DeleteComment(ctx context.Context, id string) error
	ListComments(ctx context.Context, postID string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Comment, *Cursor, error)

	// Reactions
	UpsertReaction(ctx context.Context, r *Reaction) error
	DeleteReaction(ctx context.Context, targetType, targetID, personaID string) error
	GetReaction(ctx context.Context, targetType, targetID, personaID string) (*Reaction, error)
	ListReactions(ctx context.Context, targetType, targetID string, cursor *Cursor, limit int) ([]*Reaction, *Cursor, error)
	RecomputeReactionCounts(ctx context.Context, targetType, targetID string) error

	// Media
	CreateMediaAsset(ctx context.Context, m *MediaAsset) error

	// Saves
	CreateSave(ctx context.Context, personaID, postID string) error
	DeleteSave(ctx context.Context, personaID, postID string) error
	GetSave(ctx context.Context, personaID, postID string) (bool, error)

	// Idempotency
	GetIdempotencyResponse(ctx context.Context, key, method, path string) (*IdempotencyResponse, error)
	SaveIdempotencyResponse(ctx context.Context, key, method, path string, status int, body []byte) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListTopics(ctx context.Context, query string, cursor *Cursor, limit int) ([]*Topic, *Cursor, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	q := "%" + query + "%"
	var cursorCreatedAt *time.Time
	var cursorID any
	if cursor != nil {
		t := cursor.CreatedAt
		cursorCreatedAt = &t
		cursorID = cursor.ID
	}

	const sql = `
		SELECT id, name, description, category, status, slug, icon_url, cover_url, note_count, follower_count, created_at, updated_at
		FROM topics
		WHERE status = 'active'
		  AND ($1 = '' OR name ILIKE $2 OR description ILIKE $2)
		  AND ($3::timestamptz IS NULL OR created_at > $3 OR (created_at = $3 AND id > $4::uuid))
		ORDER BY created_at ASC, id ASC
		LIMIT $5
	`
	rows, err := r.pool.Query(ctx, sql, query, q, cursorCreatedAt, cursorID, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list topics: %w", err)
	}
	defer rows.Close()

	var out []*Topic
	for rows.Next() {
		t, err := scanTopic(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate topics: %w", err)
	}

	var next *Cursor
	if len(out) > limit {
		last := out[limit-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		out = out[:limit]
	}
	return out, next, nil
}

func (r *PostgresRepository) GetTopic(ctx context.Context, id string) (*Topic, error) {
	const sql = `
		SELECT id, name, description, category, status, slug, icon_url, cover_url, note_count, follower_count, created_at, updated_at
		FROM topics
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, sql, id)
	return scanTopic(row)
}

func scanTopic(row pgx.Row) (*Topic, error) {
	var t Topic
	if err := row.Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&t.Category,
		&t.Status,
		&t.Slug,
		&t.IconURL,
		&t.CoverURL,
		&t.NoteCount,
		&t.FollowerCount,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTopicNotFound
		}
		return nil, fmt.Errorf("scan topic: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) GetTopicFollow(ctx context.Context, personaID, topicID string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM topic_follows WHERE persona_id = $1 AND topic_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, sql, personaID, topicID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check topic follow: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) FollowTopic(ctx context.Context, personaID, topicID string) error {
	const sql = `
		INSERT INTO topic_follows (persona_id, topic_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	tag, err := r.pool.Exec(ctx, sql, personaID, topicID)
	if err != nil {
		return fmt.Errorf("follow topic: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTopicAlreadyFollowed
	}
	return nil
}

func (r *PostgresRepository) UnfollowTopic(ctx context.Context, personaID, topicID string) error {
	const sql = `DELETE FROM topic_follows WHERE persona_id = $1 AND topic_id = $2`
	tag, err := r.pool.Exec(ctx, sql, personaID, topicID)
	if err != nil {
		return fmt.Errorf("unfollow topic: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrTopicNotFollowed
	}
	return nil
}

func (r *PostgresRepository) IncrementTopicNoteCount(ctx context.Context, topicID string, delta int) error {
	const sql = `
		UPDATE topics
		SET note_count = GREATEST(0, note_count + $2), updated_at = now()
		WHERE id = $1
	`
	if _, err := r.pool.Exec(ctx, sql, topicID, delta); err != nil {
		return fmt.Errorf("increment topic note count: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreatePost(ctx context.Context, p *Post) error {
	const sql = `
		INSERT INTO posts (persona_id, topic_id, content, moderation_state, reaction_counts, reply_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	rc, err := json.Marshal(p.ReactionCounts)
	if err != nil {
		return fmt.Errorf("marshal reaction counts: %w", err)
	}
	row := r.pool.QueryRow(ctx, sql, p.PersonaID, p.TopicID, p.Content, p.ModerationState, rc, p.ReplyCount)
	if err := row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return fmt.Errorf("insert post: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetPost(ctx context.Context, id string) (*Post, error) {
	const sql = `
		SELECT id, persona_id, topic_id, content, moderation_state, reaction_counts, reply_count, created_at, updated_at, deleted_at
		FROM posts
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, sql, id)
	return scanPost(row)
}

func (r *PostgresRepository) GetPostByIDAndPersona(ctx context.Context, id, personaID string) (*Post, error) {
	const sql = `
		SELECT id, persona_id, topic_id, content, moderation_state, reaction_counts, reply_count, created_at, updated_at, deleted_at
		FROM posts
		WHERE id = $1 AND persona_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, id, personaID)
	return scanPost(row)
}

func scanPost(row pgx.Row) (*Post, error) {
	var p Post
	var rc []byte
	if err := row.Scan(
		&p.ID,
		&p.PersonaID,
		&p.TopicID,
		&p.Content,
		&p.ModerationState,
		&rc,
		&p.ReplyCount,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("scan post: %w", err)
	}
	p.ReactionCounts = map[string]int{}
	if len(rc) > 0 {
		_ = json.Unmarshal(rc, &p.ReactionCounts)
	}
	return &p, nil
}

func (r *PostgresRepository) UpdatePost(ctx context.Context, p *Post) error {
	const sql = `
		UPDATE posts
		SET topic_id = $2,
		    content = $3,
		    moderation_state = $4,
		    updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	if _, err := r.pool.Exec(ctx, sql, p.ID, p.TopicID, p.Content, p.ModerationState); err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeletePost(ctx context.Context, id string) error {
	const sql = `
		UPDATE posts
		SET moderation_state = 'deleted', deleted_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPostNotFound
		}
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListPosts(ctx context.Context, topicID *string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error) {
	return r.listPostsInternal(ctx, topicID, nil, viewerPersonaID, cursor, limit)
}

func (r *PostgresRepository) ListTopicPosts(ctx context.Context, topicID string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error) {
	return r.listPostsInternal(ctx, &topicID, nil, viewerPersonaID, cursor, limit)
}

func (r *PostgresRepository) ListPersonaPosts(ctx context.Context, personaID string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error) {
	return r.listPostsInternal(ctx, nil, &personaID, viewerPersonaID, cursor, limit)
}

func (r *PostgresRepository) listPostsInternal(ctx context.Context, topicID, personaID *string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Post, *Cursor, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var cursorCreatedAt *time.Time
	var cursorID any
	if cursor != nil {
		t := cursor.CreatedAt
		cursorCreatedAt = &t
		cursorID = cursor.ID
	}

	const sql = `
		SELECT p.id, p.persona_id, p.topic_id, p.content, p.moderation_state, p.reaction_counts, p.reply_count, p.created_at, p.updated_at, p.deleted_at
		FROM posts p
		WHERE p.moderation_state != 'deleted'
		  AND (
		      p.moderation_state = 'published'
		      OR ($6::uuid IS NOT NULL AND p.persona_id = $6::uuid)
		  )
		  AND ($1::uuid IS NULL OR p.topic_id = $1)
		  AND ($2::uuid IS NULL OR p.persona_id = $2)
		  AND ($3::timestamptz IS NULL OR p.created_at < $3 OR (p.created_at = $3 AND p.id < $4::uuid))
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $5
	`
	rows, err := r.pool.Query(ctx, sql, topicID, personaID, cursorCreatedAt, cursorID, limit+1, viewerPersonaID)
	if err != nil {
		return nil, nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	var out []*Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate posts: %w", err)
	}

	var next *Cursor
	if len(out) > limit {
		last := out[limit-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		out = out[:limit]
	}
	return out, next, nil
}

func (r *PostgresRepository) IncrementPostReplyCount(ctx context.Context, postID string, delta int) error {
	const sql = `
		UPDATE posts
		SET reply_count = GREATEST(0, reply_count + $2), updated_at = now()
		WHERE id = $1
	`
	if _, err := r.pool.Exec(ctx, sql, postID, delta); err != nil {
		return fmt.Errorf("increment post reply count: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateComment(ctx context.Context, c *Comment) error {
	const sql = `
		INSERT INTO comments (post_id, persona_id, content, moderation_state, reaction_counts)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	rc, err := json.Marshal(c.ReactionCounts)
	if err != nil {
		return fmt.Errorf("marshal reaction counts: %w", err)
	}
	row := r.pool.QueryRow(ctx, sql, c.PostID, c.PersonaID, c.Content, c.ModerationState, rc)
	if err := row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetComment(ctx context.Context, id string) (*Comment, error) {
	const sql = `
		SELECT id, post_id, persona_id, content, moderation_state, reaction_counts, created_at, updated_at, deleted_at
		FROM comments
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, sql, id)
	return scanComment(row)
}

func (r *PostgresRepository) GetCommentByIDAndPersona(ctx context.Context, id, personaID string) (*Comment, error) {
	const sql = `
		SELECT id, post_id, persona_id, content, moderation_state, reaction_counts, created_at, updated_at, deleted_at
		FROM comments
		WHERE id = $1 AND persona_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, id, personaID)
	return scanComment(row)
}

func scanComment(row pgx.Row) (*Comment, error) {
	var c Comment
	var rc []byte
	if err := row.Scan(
		&c.ID,
		&c.PostID,
		&c.PersonaID,
		&c.Content,
		&c.ModerationState,
		&rc,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.DeletedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("scan comment: %w", err)
	}
	c.ReactionCounts = map[string]int{}
	if len(rc) > 0 {
		_ = json.Unmarshal(rc, &c.ReactionCounts)
	}
	return &c, nil
}

func (r *PostgresRepository) UpdateComment(ctx context.Context, c *Comment) error {
	const sql = `
		UPDATE comments
		SET content = $2,
		    moderation_state = $3,
		    updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	if _, err := r.pool.Exec(ctx, sql, c.ID, c.Content, c.ModerationState); err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteComment(ctx context.Context, id string) error {
	const sql = `
		UPDATE comments
		SET moderation_state = 'deleted', deleted_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrCommentNotFound
		}
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListComments(ctx context.Context, postID string, viewerPersonaID *string, cursor *Cursor, limit int) ([]*Comment, *Cursor, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var cursorCreatedAt *time.Time
	var cursorID any
	if cursor != nil {
		t := cursor.CreatedAt
		cursorCreatedAt = &t
		cursorID = cursor.ID
	}

	const sql = `
		SELECT c.id, c.post_id, c.persona_id, c.content, c.moderation_state, c.reaction_counts, c.created_at, c.updated_at, c.deleted_at
		FROM comments c
		WHERE c.post_id = $1
		  AND c.moderation_state != 'deleted'
		  AND (
		      c.moderation_state = 'published'
		      OR ($5::uuid IS NOT NULL AND c.persona_id = $5::uuid)
		  )
		  AND ($2::timestamptz IS NULL OR c.created_at < $2 OR (c.created_at = $2 AND c.id < $3::uuid))
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, sql, postID, cursorCreatedAt, cursorID, limit+1, viewerPersonaID)
	if err != nil {
		return nil, nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var out []*Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate comments: %w", err)
	}

	var next *Cursor
	if len(out) > limit {
		last := out[limit-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		out = out[:limit]
	}
	return out, next, nil
}

func (r *PostgresRepository) UpsertReaction(ctx context.Context, re *Reaction) error {
	const sql = `
		INSERT INTO reactions (target_type, target_id, persona_id, type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (target_type, target_id, persona_id) DO UPDATE
		SET type = EXCLUDED.type, updated_at = now()
		RETURNING id, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, sql, re.TargetType, re.TargetID, re.PersonaID, re.Type)
	if err := row.Scan(&re.ID, &re.CreatedAt, &re.UpdatedAt); err != nil {
		return fmt.Errorf("upsert reaction: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteReaction(ctx context.Context, targetType, targetID, personaID string) error {
	const sql = `DELETE FROM reactions WHERE target_type = $1 AND target_id = $2 AND persona_id = $3`
	tag, err := r.pool.Exec(ctx, sql, targetType, targetID, personaID)
	if err != nil {
		return fmt.Errorf("delete reaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrReactionNotFound
	}
	return nil
}

func (r *PostgresRepository) GetReaction(ctx context.Context, targetType, targetID, personaID string) (*Reaction, error) {
	const sql = `
		SELECT id, target_type, target_id, persona_id, type, created_at, updated_at
		FROM reactions
		WHERE target_type = $1 AND target_id = $2 AND persona_id = $3
	`
	row := r.pool.QueryRow(ctx, sql, targetType, targetID, personaID)
	return scanReaction(row)
}

func (r *PostgresRepository) ListReactions(ctx context.Context, targetType, targetID string, cursor *Cursor, limit int) ([]*Reaction, *Cursor, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var cursorCreatedAt *time.Time
	var cursorID any
	if cursor != nil {
		t := cursor.CreatedAt
		cursorCreatedAt = &t
		cursorID = cursor.ID
	}

	const sql = `
		SELECT id, target_type, target_id, persona_id, type, created_at, updated_at
		FROM reactions
		WHERE target_type = $1 AND target_id = $2
		  AND ($3::timestamptz IS NULL OR created_at < $3 OR (created_at = $3 AND id < $4::uuid))
		ORDER BY created_at DESC, id DESC
		LIMIT $5
	`
	rows, err := r.pool.Query(ctx, sql, targetType, targetID, cursorCreatedAt, cursorID, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list reactions: %w", err)
	}
	defer rows.Close()

	var out []*Reaction
	for rows.Next() {
		re, err := scanReaction(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, re)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate reactions: %w", err)
	}

	var next *Cursor
	if len(out) > limit {
		last := out[limit-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		out = out[:limit]
	}
	return out, next, nil
}

func scanReaction(row pgx.Row) (*Reaction, error) {
	var re Reaction
	if err := row.Scan(
		&re.ID,
		&re.TargetType,
		&re.TargetID,
		&re.PersonaID,
		&re.Type,
		&re.CreatedAt,
		&re.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrReactionNotFound
		}
		return nil, fmt.Errorf("scan reaction: %w", err)
	}
	return &re, nil
}

func (r *PostgresRepository) RecomputeReactionCounts(ctx context.Context, targetType, targetID string) error {
	const recomputeSQL = `
		WITH counts AS (
			SELECT type, COUNT(*)::int AS cnt
			FROM reactions
			WHERE target_type = $1 AND target_id = $2
			GROUP BY type
		)
		SELECT COALESCE(jsonb_object_agg(type, cnt), '{}') FROM counts
	`
	var counts []byte
	if err := r.pool.QueryRow(ctx, recomputeSQL, targetType, targetID).Scan(&counts); err != nil {
		return fmt.Errorf("recompute reaction counts: %w", err)
	}

	var updateSQL string
	if targetType == "post" {
		updateSQL = `UPDATE posts SET reaction_counts = $2, updated_at = now() WHERE id = $1`
	} else {
		updateSQL = `UPDATE comments SET reaction_counts = $2, updated_at = now() WHERE id = $1`
	}
	if _, err := r.pool.Exec(ctx, updateSQL, targetID, counts); err != nil {
		return fmt.Errorf("update reaction counts: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateMediaAsset(ctx context.Context, m *MediaAsset) error {
	const sql = `
		INSERT INTO media_assets (persona_id, url, mime_type, width, height, file_size, checksum, thumbnail_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	row := r.pool.QueryRow(ctx, sql, m.PersonaID, m.URL, m.MimeType, m.Width, m.Height, m.FileSize, m.Checksum, m.ThumbnailURL, m.Status)
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		return fmt.Errorf("insert media asset: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateSave(ctx context.Context, personaID, postID string) error {
	const sql = `
		INSERT INTO saves (persona_id, post_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	tag, err := r.pool.Exec(ctx, sql, personaID, postID)
	if err != nil {
		return fmt.Errorf("create save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSaveAlreadyExists
	}
	return nil
}

func (r *PostgresRepository) DeleteSave(ctx context.Context, personaID, postID string) error {
	const sql = `DELETE FROM saves WHERE persona_id = $1 AND post_id = $2`
	tag, err := r.pool.Exec(ctx, sql, personaID, postID)
	if err != nil {
		return fmt.Errorf("delete save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSaveNotFound
	}
	return nil
}

func (r *PostgresRepository) GetSave(ctx context.Context, personaID, postID string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM saves WHERE persona_id = $1 AND post_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, sql, personaID, postID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check save: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) GetIdempotencyResponse(ctx context.Context, key, method, path string) (*IdempotencyResponse, error) {
	const sql = `
		SELECT response_status, response_body
		FROM idempotency_keys
		WHERE key = $1
		  AND request_method = $2
		  AND request_path = $3
		  AND created_at > now() - interval '24 hours'
	`
	var resp IdempotencyResponse
	if err := r.pool.QueryRow(ctx, sql, key, method, path).Scan(&resp.Status, &resp.Body); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get idempotency response: %w", err)
	}
	return &resp, nil
}

func (r *PostgresRepository) SaveIdempotencyResponse(ctx context.Context, key, method, path string, status int, body []byte) error {
	const sql = `
		INSERT INTO idempotency_keys (key, request_method, request_path, response_status, response_body)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key, request_method, request_path) DO UPDATE
		SET response_status = EXCLUDED.response_status,
		    response_body = EXCLUDED.response_body,
		    created_at = now()
	`
	if _, err := r.pool.Exec(ctx, sql, key, method, path, status, body); err != nil {
		return fmt.Errorf("save idempotency response: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
