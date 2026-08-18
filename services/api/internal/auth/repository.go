package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// EmailCode stores a hashed verification code.
type EmailCode struct {
	ID          string
	UserID      *string
	Email       string
	CodeHash    string
	Purpose     string
	Attempts    int
	MaxAttempts int
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
	IPAddress   string
	Fingerprint string
}

// Session stores a refresh-token hash and metadata.
type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	LastUsedAt       time.Time
	IPAddress        string
	UserAgent        string
	Fingerprint      string
}

// InviteCode is a single-use registration invite.
type InviteCode struct {
	ID        string
	Code      string
	CreatedBy string
	UsedBy    *string
	UsedAt    *time.Time
	ExpiresAt *time.Time
	CreatedAt time.Time
}

// AuditEvent records security-relevant occurrences.
type AuditEvent struct {
	ID          string
	UserID      *string
	SessionID   *string
	EventType   string
	IPAddress   string
	UserAgent   string
	Fingerprint string
	Metadata    map[string]any
	CreatedAt   time.Time
}

// Repository provides persistence for auth entities.
type Repository interface {
	CreateUser(ctx context.Context, username, passwordHash string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	MarkUserDeleting(ctx context.Context, id string, gracePeriod time.Duration) error
	ListUsersPastGracePeriod(ctx context.Context, now time.Time) ([]*User, error)
	MarkUserDeleted(ctx context.Context, id string) error

	CreateEmailCode(ctx context.Context, code *EmailCode) error
	GetActiveEmailCode(ctx context.Context, email, purpose string) (*EmailCode, error)
	MarkEmailCodeUsed(ctx context.Context, id string) error
	IncrementEmailCodeAttempts(ctx context.Context, id string) error

	CreateSession(ctx context.Context, session *Session) error
	GetSessionByRefreshHash(ctx context.Context, hash string) (*Session, error)
	RevokeSession(ctx context.Context, id string) error
	RevokeAllSessionsForUser(ctx context.Context, userID string) error
	UpdateSessionLastUsed(ctx context.Context, id string) error

	CreateAuditEvent(ctx context.Context, event *AuditEvent) error

	CreateInviteCode(ctx context.Context, code *InviteCode) error
	GetInviteCode(ctx context.Context, code string) (*InviteCode, error)
	GetInviteCodeByID(ctx context.Context, id string) (*InviteCode, error)
	ListInviteCodes(ctx context.Context, cursor *time.Time, limit int) ([]*InviteCode, *time.Time, error)
	DeleteInviteCode(ctx context.Context, id string) error
	CreateUserAndConsumeInviteCode(ctx context.Context, username, passwordHash, inviteCode string) (*User, error)
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// NormalizeEmail lowercases and trims whitespace from an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

const userColumns = `id, COALESCE(username, '') AS username, COALESCE(password_hash, '') AS password_hash, COALESCE(email_normalized, '') AS email_normalized, status, deletion_requested_at, deletion_grace_period_ends_at, created_at, updated_at`

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
	u, err := scanUser(r.pool.QueryRow(ctx, sql, username, passwordHash))
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
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

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	const sql = `
		SELECT ` + userColumns + `
		FROM users
		WHERE id = $1
	`
	u, err := scanUser(r.pool.QueryRow(ctx, sql, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("select user by id: %w", err)
	}
	return u, nil
}

func (r *PostgresRepository) MarkUserDeleting(ctx context.Context, id string, gracePeriod time.Duration) error {
	const sql = `
		UPDATE users
		SET status = 'deleting',
		    deletion_requested_at = COALESCE(deletion_requested_at, now()),
		    deletion_grace_period_ends_at = COALESCE(deletion_grace_period_ends_at, now() + $2::interval),
		    updated_at = now()
		WHERE id = $1
		RETURNING status
	`
	var status string
	if err := r.pool.QueryRow(ctx, sql, id, gracePeriod).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("mark user deleting: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListUsersPastGracePeriod(ctx context.Context, now time.Time) ([]*User, error) {
	const sql = `
		SELECT ` + userColumns + `
		FROM users
		WHERE status = 'deleting' AND deletion_grace_period_ends_at IS NOT NULL AND deletion_grace_period_ends_at <= $1
	`
	rows, err := r.pool.Query(ctx, sql, now)
	if err != nil {
		return nil, fmt.Errorf("list users past grace period: %w", err)
	}
	defer rows.Close()

	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users past grace period: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) MarkUserDeleted(ctx context.Context, id string) error {
	const sql = `
		UPDATE users
		SET status = 'deleted',
		    email_normalized = gen_random_uuid()::text,
		    updated_at = now()
		WHERE id = $1
		RETURNING status
	`
	var status string
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("mark user deleted: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateEmailCode(ctx context.Context, code *EmailCode) error {
	const sql = `
		INSERT INTO email_codes (user_id, email, code_hash, purpose, attempts, max_attempts, expires_at, ip_address, fingerprint)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	row := r.pool.QueryRow(ctx, sql,
		code.UserID,
		code.Email,
		code.CodeHash,
		code.Purpose,
		code.Attempts,
		code.MaxAttempts,
		code.ExpiresAt,
		code.IPAddress,
		code.Fingerprint,
	)
	if err := row.Scan(&code.ID, &code.CreatedAt); err != nil {
		return fmt.Errorf("insert email code: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetActiveEmailCode(ctx context.Context, email, purpose string) (*EmailCode, error) {
	const sql = `
		SELECT id, user_id, email, code_hash, purpose, attempts, max_attempts, expires_at, used_at, created_at, ip_address, fingerprint
		FROM email_codes
		WHERE email = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, sql, NormalizeEmail(email), purpose)
	var c EmailCode
	if err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Email,
		&c.CodeHash,
		&c.Purpose,
		&c.Attempts,
		&c.MaxAttempts,
		&c.ExpiresAt,
		&c.UsedAt,
		&c.CreatedAt,
		&c.IPAddress,
		&c.Fingerprint,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("select email code: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) MarkEmailCodeUsed(ctx context.Context, id string) error {
	const sql = `UPDATE email_codes SET used_at = now() WHERE id = $1 AND used_at IS NULL RETURNING id`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("code already used or not found")
		}
		return fmt.Errorf("mark email code used: %w", err)
	}
	return nil
}

func (r *PostgresRepository) IncrementEmailCodeAttempts(ctx context.Context, id string) error {
	const sql = `
		UPDATE email_codes
		SET attempts = attempts + 1
		WHERE id = $1
		RETURNING attempts, max_attempts
	`
	var attempts, maxAttempts int
	if err := r.pool.QueryRow(ctx, sql, id).Scan(&attempts, &maxAttempts); err != nil {
		return fmt.Errorf("increment attempts: %w", err)
	}
	if attempts >= maxAttempts {
		// Exhausting attempts invalidates the code by marking it used.
		if err := r.MarkEmailCodeUsed(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session *Session) error {
	const sql = `
		INSERT INTO sessions (user_id, refresh_token_hash, expires_at, ip_address, user_agent, fingerprint)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, last_used_at
	`
	row := r.pool.QueryRow(ctx, sql,
		session.UserID,
		session.RefreshTokenHash,
		session.ExpiresAt,
		session.IPAddress,
		session.UserAgent,
		session.Fingerprint,
	)
	if err := row.Scan(&session.ID, &session.CreatedAt, &session.LastUsedAt); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetSessionByRefreshHash(ctx context.Context, hash string) (*Session, error) {
	const sql = `
		SELECT id, user_id, refresh_token_hash, created_at, expires_at, revoked_at, last_used_at, ip_address, user_agent, fingerprint
		FROM sessions
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
	`
	row := r.pool.QueryRow(ctx, sql, hash)
	var s Session
	if err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.RefreshTokenHash,
		&s.CreatedAt,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.LastUsedAt,
		&s.IPAddress,
		&s.UserAgent,
		&s.Fingerprint,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("select session by refresh hash: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, id string) error {
	const sql = `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`
	if _, err := r.pool.Exec(ctx, sql, id); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RevokeAllSessionsForUser(ctx context.Context, userID string) error {
	const sql = `UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := r.pool.Exec(ctx, sql, userID); err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateSessionLastUsed(ctx context.Context, id string) error {
	const sql = `UPDATE sessions SET last_used_at = now() WHERE id = $1`
	if _, err := r.pool.Exec(ctx, sql, id); err != nil {
		return fmt.Errorf("update session last used: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateAuditEvent(ctx context.Context, event *AuditEvent) error {
	const sql = `
		INSERT INTO audit_events (user_id, session_id, event_type, ip_address, user_agent, fingerprint, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	row := r.pool.QueryRow(ctx, sql,
		event.UserID,
		event.SessionID,
		event.EventType,
		event.IPAddress,
		event.UserAgent,
		event.Fingerprint,
		event.Metadata,
	)
	if err := row.Scan(&event.ID, &event.CreatedAt); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

const inviteCodeColumns = `id, code, created_by, used_by, used_at, expires_at, created_at`

func scanInviteCode(row pgx.Row) (*InviteCode, error) {
	var c InviteCode
	if err := row.Scan(
		&c.ID,
		&c.Code,
		&c.CreatedBy,
		&c.UsedBy,
		&c.UsedAt,
		&c.ExpiresAt,
		&c.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepository) CreateInviteCode(ctx context.Context, code *InviteCode) error {
	const sql = `
		INSERT INTO invite_codes (code, created_by, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	row := r.pool.QueryRow(ctx, sql, code.Code, code.CreatedBy, code.ExpiresAt)
	if err := row.Scan(&code.ID, &code.CreatedAt); err != nil {
		return fmt.Errorf("insert invite code: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetInviteCode(ctx context.Context, code string) (*InviteCode, error) {
	const sql = `
		SELECT ` + inviteCodeColumns + `
		FROM invite_codes
		WHERE code = $1
	`
	c, err := scanInviteCode(r.pool.QueryRow(ctx, sql, code))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select invite code: %w", err)
	}
	return c, nil
}

func (r *PostgresRepository) GetInviteCodeByID(ctx context.Context, id string) (*InviteCode, error) {
	const sql = `
		SELECT ` + inviteCodeColumns + `
		FROM invite_codes
		WHERE id = $1
	`
	c, err := scanInviteCode(r.pool.QueryRow(ctx, sql, id))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select invite code by id: %w", err)
	}
	return c, nil
}

func (r *PostgresRepository) ListInviteCodes(ctx context.Context, cursor *time.Time, limit int) ([]*InviteCode, *time.Time, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	const sql = `
		SELECT ` + inviteCodeColumns + `
		FROM invite_codes
		WHERE ($1::timestamptz IS NULL OR created_at < $1)
		ORDER BY created_at DESC, id DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, sql, cursor, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list invite codes: %w", err)
	}
	defer rows.Close()

	var out []*InviteCode
	for rows.Next() {
		c, err := scanInviteCode(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate invite codes: %w", err)
	}

	var next *time.Time
	if len(out) > limit {
		last := out[limit-1]
		t := last.CreatedAt
		next = &t
		out = out[:limit]
	}
	return out, next, nil
}

func (r *PostgresRepository) DeleteInviteCode(ctx context.Context, id string) error {
	const sql = `DELETE FROM invite_codes WHERE id = $1`
	if _, err := r.pool.Exec(ctx, sql, id); err != nil {
		return fmt.Errorf("delete invite code: %w", err)
	}
	return nil
}

// CreateUserAndConsumeInviteCode inserts the user and marks the invite code used
// in a single transaction. A concurrent registration with the same code loses
// the UPDATE and causes the whole transaction (user insert included) to roll back.
func (r *PostgresRepository) CreateUserAndConsumeInviteCode(ctx context.Context, username, passwordHash, inviteCode string) (*User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertUser = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING ` + userColumns
	u, err := scanUser(tx.QueryRow(ctx, insertUser, username, passwordHash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrUsernameTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	const consumeInvite = `
		UPDATE invite_codes
		SET used_by = $2, used_at = now()
		WHERE code = $1 AND used_by IS NULL AND (expires_at IS NULL OR expires_at > now())
	`
	tag, err := tx.Exec(ctx, consumeInvite, inviteCode, u.ID)
	if err != nil {
		return nil, fmt.Errorf("consume invite code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrInviteCodeUsed
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return u, nil
}
