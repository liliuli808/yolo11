package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Domain errors used by the identity subsystem.
var (
	ErrProfileNotFound        = errors.New("profile not found")
	ErrInvalidEmail           = errors.New("invalid email")
	ErrEmailAlreadyUsed       = errors.New("email already used")
	ErrInvalidCode            = errors.New("invalid or expired verification code")
	ErrInvalidExportFormat    = errors.New("invalid export format")
	ErrExportNotFound         = errors.New("export not found")
	ErrExportRateLimited      = errors.New("export rate limited")
	ErrPersonaNotFound        = errors.New("persona not found")
	ErrPersonaNotOwned        = errors.New("persona not owned")
	ErrPersonaMaxReached      = errors.New("max personas reached")
	ErrPersonaAliasTaken      = errors.New("alias taken")
	ErrPersonaAliasDisallowed = errors.New("alias disallowed")
	ErrPersonaArchived        = errors.New("persona archived")
	ErrPersonaRestricted      = errors.New("persona restricted")
)

// RealProfile is the account root returned to its owner.
type RealProfile struct {
	ID                        string
	EmailNormalized           string
	Status                    string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	DefaultPersonaID          *string
	PersonaCount              int
	MaxPersonas               int
	DeletionRequestedAt       *time.Time
	DeletionGracePeriodEndsAt *time.Time
}

// Persona is an anonymous public identity owned by a real profile.
type Persona struct {
	ID            string
	RealProfileID string
	Alias         string
	Bio           *string
	AvatarSeed    string
	AvatarColor   string
	Status        string
	IsDefault     bool
	NoteCount     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ArchivedAt    *time.Time
}

// DataExport is a request for an archive of a real profile's data.
type DataExport struct {
	ID            string
	RealProfileID string
	Status        string
	Format        string
	RequestedAt   time.Time
	ReadyAt       *time.Time
	DownloadURL   *string
	ExpiresAt     *time.Time
}

// PersonaCreateRequest contains the fields accepted when creating a persona.
type PersonaCreateRequest struct {
	Alias       string
	Bio         *string
	AvatarSeed  string
	AvatarColor string
}

// PersonaUpdateRequest contains the optional fields accepted when updating a persona.
type PersonaUpdateRequest struct {
	Alias       *string
	Bio         *string
	AvatarSeed  *string
	AvatarColor *string
}

// Repository provides persistence for identity entities.
type Repository interface {
	GetRealProfileByID(ctx context.Context, id string) (*RealProfile, error)
	UpdateEmail(ctx context.Context, id, email string) error
	EmailExistsForOther(ctx context.Context, id, email string) (bool, error)

	CountActivePersonas(ctx context.Context, realProfileID string) (int, error)
	ListPersonasByRealProfile(ctx context.Context, realProfileID string) ([]*Persona, error)
	GetPersonaByID(ctx context.Context, id string) (*Persona, error)
	GetPersonaByIDAndRealProfile(ctx context.Context, id, realProfileID string) (*Persona, error)
	CreatePersona(ctx context.Context, p *Persona) error
	UpdatePersona(ctx context.Context, p *Persona) error
	ArchivePersona(ctx context.Context, id, realProfileID string) error
	SetDefaultPersona(ctx context.Context, id, realProfileID string) error
	ClearDefaultPersonaForRealProfile(ctx context.Context, realProfileID string) error
	ArchiveAllPersonasForRealProfile(ctx context.Context, realProfileID string) error

	CreateDataExport(ctx context.Context, e *DataExport) error
	GetDataExportByID(ctx context.Context, id, realProfileID string) (*DataExport, error)
	GetLatestDataExport(ctx context.Context, realProfileID string) (*DataExport, error)
	ListPendingDataExports(ctx context.Context, olderThan time.Time) ([]*DataExport, error)
	UpdateDataExportReady(ctx context.Context, id string, downloadURL string, expiresAt time.Time) error
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetRealProfileByID(ctx context.Context, id string) (*RealProfile, error) {
	const sql = `
		SELECT
			u.id,
			COALESCE(u.email_normalized, '') AS email_normalized,
			u.status,
			u.created_at,
			u.updated_at,
			u.default_persona_id,
			u.max_personas,
			u.deletion_requested_at,
			u.deletion_grace_period_ends_at,
			COUNT(p.id) FILTER (WHERE p.status = 'active')
		FROM users u
		LEFT JOIN personas p ON p.real_profile_id = u.id
		WHERE u.id = $1
		GROUP BY u.id
	`
	row := r.pool.QueryRow(ctx, sql, id)
	var profile RealProfile
	if err := row.Scan(
		&profile.ID,
		&profile.EmailNormalized,
		&profile.Status,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&profile.DefaultPersonaID,
		&profile.MaxPersonas,
		&profile.DeletionRequestedAt,
		&profile.DeletionGracePeriodEndsAt,
		&profile.PersonaCount,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("select real profile: %w", err)
	}
	return &profile, nil
}

func (r *PostgresRepository) UpdateEmail(ctx context.Context, id, email string) error {
	const sql = `UPDATE users SET email_normalized = $2, updated_at = now() WHERE id = $1`
	if _, err := r.pool.Exec(ctx, sql, id, email); err != nil {
		return fmt.Errorf("update email: %w", err)
	}
	return nil
}

func (r *PostgresRepository) EmailExistsForOther(ctx context.Context, id, email string) (bool, error) {
	const sql = `SELECT EXISTS(SELECT 1 FROM users WHERE email_normalized = $2 AND id <> $1)`
	var exists bool
	if err := r.pool.QueryRow(ctx, sql, id, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return exists, nil
}

func (r *PostgresRepository) CountActivePersonas(ctx context.Context, realProfileID string) (int, error) {
	const sql = `SELECT COUNT(*) FROM personas WHERE real_profile_id = $1 AND status = 'active'`
	var count int
	if err := r.pool.QueryRow(ctx, sql, realProfileID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active personas: %w", err)
	}
	return count, nil
}

func (r *PostgresRepository) ListPersonasByRealProfile(ctx context.Context, realProfileID string) ([]*Persona, error) {
	const sql = `
		SELECT id, real_profile_id, alias, bio, avatar_seed, avatar_color, status, is_default, note_count, created_at, updated_at, archived_at
		FROM personas
		WHERE real_profile_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, sql, realProfileID)
	if err != nil {
		return nil, fmt.Errorf("list personas: %w", err)
	}
	defer rows.Close()

	var out []*Persona
	for rows.Next() {
		p, err := scanPersona(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate personas: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) GetPersonaByID(ctx context.Context, id string) (*Persona, error) {
	const sql = `
		SELECT id, real_profile_id, alias, bio, avatar_seed, avatar_color, status, is_default, note_count, created_at, updated_at, archived_at
		FROM personas
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, sql, id)
	return scanPersona(row)
}

func (r *PostgresRepository) GetPersonaByIDAndRealProfile(ctx context.Context, id, realProfileID string) (*Persona, error) {
	const sql = `
		SELECT id, real_profile_id, alias, bio, avatar_seed, avatar_color, status, is_default, note_count, created_at, updated_at, archived_at
		FROM personas
		WHERE id = $1 AND real_profile_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, id, realProfileID)
	return scanPersona(row)
}

type personaRow interface {
	Scan(dest ...any) error
}

func scanPersona(row personaRow) (*Persona, error) {
	var p Persona
	if err := row.Scan(
		&p.ID,
		&p.RealProfileID,
		&p.Alias,
		&p.Bio,
		&p.AvatarSeed,
		&p.AvatarColor,
		&p.Status,
		&p.IsDefault,
		&p.NoteCount,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.ArchivedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPersonaNotFound
		}
		return nil, fmt.Errorf("scan persona: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) CreatePersona(ctx context.Context, p *Persona) error {
	const sql = `
		INSERT INTO personas (real_profile_id, alias, bio, avatar_seed, avatar_color, status, is_default, note_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, sql,
		p.RealProfileID,
		p.Alias,
		p.Bio,
		p.AvatarSeed,
		p.AvatarColor,
		p.Status,
		p.IsDefault,
		p.NoteCount,
	)
	if err := row.Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return ErrPersonaAliasTaken
		}
		return fmt.Errorf("insert persona: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdatePersona(ctx context.Context, p *Persona) error {
	const sql = `
		UPDATE personas
		SET alias = COALESCE($2, alias),
		    bio = CASE WHEN $3 IS NULL THEN bio ELSE $3 END,
		    avatar_seed = COALESCE($4, avatar_seed),
		    avatar_color = COALESCE($5, avatar_color),
		    updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	var bio *string
	if p.Bio != nil {
		bio = p.Bio
	}
	row := r.pool.QueryRow(ctx, sql, p.ID, p.Alias, bio, p.AvatarSeed, p.AvatarColor)
	if err := row.Scan(&p.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPersonaNotFound
		}
		if isUniqueViolation(err) {
			return ErrPersonaAliasTaken
		}
		return fmt.Errorf("update persona: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ArchivePersona(ctx context.Context, id, realProfileID string) error {
	const sql = `
		UPDATE personas
		SET status = 'archived', is_default = false, archived_at = now(), updated_at = now()
		WHERE id = $1 AND real_profile_id = $2 AND status != 'archived'
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, realProfileID).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPersonaNotFound
		}
		return fmt.Errorf("archive persona: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetDefaultPersona(ctx context.Context, id, realProfileID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const clearSQL = `
		UPDATE personas
		SET is_default = false, updated_at = now()
		WHERE real_profile_id = $1 AND is_default = true
	`
	if _, err := tx.Exec(ctx, clearSQL, realProfileID); err != nil {
		return fmt.Errorf("clear default persona: %w", err)
	}

	const setSQL = `
		UPDATE personas
		SET is_default = true, updated_at = now()
		WHERE id = $1 AND real_profile_id = $2 AND status = 'active'
		RETURNING id
	`
	var returnedID string
	if err := tx.QueryRow(ctx, setSQL, id, realProfileID).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPersonaNotFound
		}
		return fmt.Errorf("set default persona: %w", err)
	}

	const updateUserSQL = `UPDATE users SET default_persona_id = $2, updated_at = now() WHERE id = $1`
	if _, err := tx.Exec(ctx, updateUserSQL, realProfileID, id); err != nil {
		return fmt.Errorf("update user default persona: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit default persona: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ClearDefaultPersonaForRealProfile(ctx context.Context, realProfileID string) error {
	const sql = `UPDATE users SET default_persona_id = null, updated_at = now() WHERE id = $1`
	if _, err := r.pool.Exec(ctx, sql, realProfileID); err != nil {
		return fmt.Errorf("clear default persona: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ArchiveAllPersonasForRealProfile(ctx context.Context, realProfileID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const archiveSQL = `
		UPDATE personas
		SET status = 'archived', is_default = false, archived_at = now(), updated_at = now()
		WHERE real_profile_id = $1 AND status != 'archived'
	`
	if _, err := tx.Exec(ctx, archiveSQL, realProfileID); err != nil {
		return fmt.Errorf("archive all personas: %w", err)
	}

	const clearDefaultSQL = `UPDATE users SET default_persona_id = null, updated_at = now() WHERE id = $1`
	if _, err := tx.Exec(ctx, clearDefaultSQL, realProfileID); err != nil {
		return fmt.Errorf("clear user default persona: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit archive all: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateDataExport(ctx context.Context, e *DataExport) error {
	const sql = `
		INSERT INTO data_exports (real_profile_id, status, format)
		VALUES ($1, $2, $3)
		RETURNING id, requested_at
	`
	row := r.pool.QueryRow(ctx, sql, e.RealProfileID, e.Status, e.Format)
	if err := row.Scan(&e.ID, &e.RequestedAt); err != nil {
		return fmt.Errorf("insert data export: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetDataExportByID(ctx context.Context, id, realProfileID string) (*DataExport, error) {
	const sql = `
		SELECT id, real_profile_id, status, format, requested_at, ready_at, download_url, expires_at
		FROM data_exports
		WHERE id = $1 AND real_profile_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, id, realProfileID)
	var e DataExport
	if err := row.Scan(
		&e.ID,
		&e.RealProfileID,
		&e.Status,
		&e.Format,
		&e.RequestedAt,
		&e.ReadyAt,
		&e.DownloadURL,
		&e.ExpiresAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrExportNotFound
		}
		return nil, fmt.Errorf("select data export: %w", err)
	}
	return &e, nil
}

func (r *PostgresRepository) GetLatestDataExport(ctx context.Context, realProfileID string) (*DataExport, error) {
	const sql = `
		SELECT id, real_profile_id, status, format, requested_at, ready_at, download_url
		FROM data_exports
		WHERE real_profile_id = $1
		ORDER BY requested_at DESC
		LIMIT 1
	`
	row := r.pool.QueryRow(ctx, sql, realProfileID)
	var e DataExport
	if err := row.Scan(
		&e.ID,
		&e.RealProfileID,
		&e.Status,
		&e.Format,
		&e.RequestedAt,
		&e.ReadyAt,
		&e.DownloadURL,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("select latest data export: %w", err)
	}
	return &e, nil
}

func (r *PostgresRepository) ListPendingDataExports(ctx context.Context, olderThan time.Time) ([]*DataExport, error) {
	const sql = `
		SELECT id, real_profile_id, status, format, requested_at, ready_at, download_url
		FROM data_exports
		WHERE status = 'pending' AND requested_at <= $1
		ORDER BY requested_at ASC
	`
	rows, err := r.pool.Query(ctx, sql, olderThan)
	if err != nil {
		return nil, fmt.Errorf("list pending exports: %w", err)
	}
	defer rows.Close()

	var out []*DataExport
	for rows.Next() {
		var e DataExport
		if err := rows.Scan(
			&e.ID,
			&e.RealProfileID,
			&e.Status,
			&e.Format,
			&e.RequestedAt,
			&e.ReadyAt,
			&e.DownloadURL,
		); err != nil {
			return nil, fmt.Errorf("scan pending export: %w", err)
		}
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending exports: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) UpdateDataExportReady(ctx context.Context, id string, downloadURL string, expiresAt time.Time) error {
	const sql = `
		UPDATE data_exports
		SET status = 'ready',
		    ready_at = now(),
		    download_url = $2,
		    expires_at = $3,
		    updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, downloadURL, expiresAt).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrExportNotFound
		}
		return fmt.Errorf("mark export ready: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
