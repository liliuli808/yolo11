package moderation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Domain errors used by the moderation subsystem.
var (
	ErrBlockNotFound         = errors.New("block not found")
	ErrBlockAlreadyExists    = errors.New("block already exists")
	ErrBlockSelf             = errors.New("cannot block self")
	ErrReportNotFound        = errors.New("report not found")
	ErrReportDuplicate       = errors.New("report already submitted")
	ErrReportInvalidTarget   = errors.New("report invalid target")
	ErrReportTargetNotFound  = errors.New("report target not found")
	ErrReportSelf            = errors.New("cannot report self")
	ErrCaseNotFound          = errors.New("case not found")
	ErrCaseInvalidOutcome    = errors.New("case invalid outcome")
	ErrCaseInvalidStatus     = errors.New("case invalid status")
	ErrPersonaNotFound       = errors.New("persona not found")
	ErrPostNotFound          = errors.New("post not found")
	ErrCommentNotFound       = errors.New("comment not found")
)

// Block records a directional persona block.
type Block struct {
	ID               string
	BlockerPersonaID string
	BlockedPersonaID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Report records a user-submitted report.
type Report struct {
	ID                   string
	ReporterRealProfileID string
	TargetType           string
	TargetID             string
	Category             string
	Details              *string
	Status               string
	ModeratorNote        *string
	CreatedAt            time.Time
	ResolvedAt           *time.Time
}

// ModerationCase aggregates reports and tracks moderator review.
type ModerationCase struct {
	ID        string
	TargetType string
	TargetID  string
	Status    string
	Outcome   *string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ModerationAction records a single moderation decision or state change.
type ModerationAction struct {
	ID             string
	CaseID         *string
	ModeratorUserID *string
	ActionType     string
	TargetType     *string
	TargetID       *string
	Note           *string
	CreatedAt      time.Time
}

// Repository provides persistence for moderation entities.
type Repository interface {
	// Blocks
	CreateBlock(ctx context.Context, b *Block) error
	DeleteBlock(ctx context.Context, id, blockerPersonaID string) error
	GetBlock(ctx context.Context, id, blockerPersonaID string) (*Block, error)
	ListBlocks(ctx context.Context, blockerPersonaID string, cursor *time.Time, limit int) ([]*Block, *time.Time, error)
	ListBlockedPersonaIDs(ctx context.Context, blockerPersonaID string) ([]string, error)
	IsBlocked(ctx context.Context, personaA, personaB string) (bool, error)

	// Reports
	CreateReport(ctx context.Context, r *Report) error
	GetReportByIDAndReporter(ctx context.Context, id, reporterID string) (*Report, error)
	GetReportByID(ctx context.Context, id string) (*Report, error)
	ListReportsByTarget(ctx context.Context, targetType, targetID string) ([]*Report, error)
	MarkReportsResolvedForCase(ctx context.Context, caseID string) error

	// Cases
	CreateCase(ctx context.Context, c *ModerationCase, reportIDs []string) error
	GetCase(ctx context.Context, id string) (*ModerationCase, error)
	GetCaseReportIDs(ctx context.Context, id string) ([]string, error)
	ListCases(ctx context.Context, status *string, cursor *time.Time, limit int) ([]*ModerationCase, *time.Time, error)
	UpdateCase(ctx context.Context, c *ModerationCase) error

	// Actions / audit
	CreateAction(ctx context.Context, a *ModerationAction) error
	ListActionsByCase(ctx context.Context, caseID string) ([]*ModerationAction, error)

	// Content/account mutations for moderation actions
	UpdatePostModerationState(ctx context.Context, id, state string) error
	UpdateCommentModerationState(ctx context.Context, id, state string) error
	UpdatePersonaStatus(ctx context.Context, id, status string) error
	UpdateUserStatus(ctx context.Context, id, status string) error
	GetPostByID(ctx context.Context, id string) (*postRef, error)
	GetCommentByID(ctx context.Context, id string) (*commentRef, error)
	GetPersonaByID(ctx context.Context, id string) (*personaRef, error)
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository backed by pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

type postRef struct {
	ID        string
	PersonaID string
}

type commentRef struct {
	ID        string
	PersonaID string
}

type personaRef struct {
	ID            string
	RealProfileID string
}

func (r *PostgresRepository) CreateBlock(ctx context.Context, b *Block) error {
	const sql = `
		INSERT INTO blocks (blocker_persona_id, blocked_persona_id)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, sql, b.BlockerPersonaID, b.BlockedPersonaID)
	if err := row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return ErrBlockAlreadyExists
		}
		if isCheckViolation(err, "blocks_no_self_block") {
			return ErrBlockSelf
		}
		return fmt.Errorf("insert block: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteBlock(ctx context.Context, id, blockerPersonaID string) error {
	const sql = `
		DELETE FROM blocks
		WHERE id = $1 AND blocker_persona_id = $2
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, blockerPersonaID).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrBlockNotFound
		}
		return fmt.Errorf("delete block: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetBlock(ctx context.Context, id, blockerPersonaID string) (*Block, error) {
	const sql = `
		SELECT id, blocker_persona_id, blocked_persona_id, created_at, updated_at
		FROM blocks
		WHERE id = $1 AND blocker_persona_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, id, blockerPersonaID)
	return scanBlock(row)
}

func (r *PostgresRepository) ListBlocks(ctx context.Context, blockerPersonaID string, cursor *time.Time, limit int) ([]*Block, *time.Time, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	const sql = `
		SELECT id, blocker_persona_id, blocked_persona_id, created_at, updated_at
		FROM blocks
		WHERE blocker_persona_id = $1
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, sql, blockerPersonaID, cursor, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list blocks: %w", err)
	}
	defer rows.Close()

	var out []*Block
	for rows.Next() {
		b, err := scanBlock(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate blocks: %w", err)
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

func (r *PostgresRepository) ListBlockedPersonaIDs(ctx context.Context, blockerPersonaID string) ([]string, error) {
	const sql = `SELECT blocked_persona_id FROM blocks WHERE blocker_persona_id = $1`
	rows, err := r.pool.Query(ctx, sql, blockerPersonaID)
	if err != nil {
		return nil, fmt.Errorf("list blocked persona ids: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan blocked persona id: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocked persona ids: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) IsBlocked(ctx context.Context, personaA, personaB string) (bool, error) {
	const sql = `
		SELECT EXISTS(
			SELECT 1 FROM blocks
			WHERE (blocker_persona_id = $1 AND blocked_persona_id = $2)
			   OR (blocker_persona_id = $2 AND blocked_persona_id = $1)
		)
	`
	var exists bool
	if err := r.pool.QueryRow(ctx, sql, personaA, personaB).Scan(&exists); err != nil {
		return false, fmt.Errorf("check block: %w", err)
	}
	return exists, nil
}

func scanBlock(row pgx.Row) (*Block, error) {
	var b Block
	if err := row.Scan(&b.ID, &b.BlockerPersonaID, &b.BlockedPersonaID, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrBlockNotFound
		}
		return nil, fmt.Errorf("scan block: %w", err)
	}
	return &b, nil
}

func (r *PostgresRepository) CreateReport(ctx context.Context, rep *Report) error {
	const sql = `
		INSERT INTO reports (reporter_real_profile_id, target_type, target_id, category, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at
	`
	row := r.pool.QueryRow(ctx, sql, rep.ReporterRealProfileID, rep.TargetType, rep.TargetID, rep.Category, rep.Details)
	if err := row.Scan(&rep.ID, &rep.Status, &rep.CreatedAt); err != nil {
		return fmt.Errorf("insert report: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetReportByIDAndReporter(ctx context.Context, id, reporterID string) (*Report, error) {
	const sql = `
		SELECT id, reporter_real_profile_id, target_type, target_id, category, details, status, moderator_note, created_at, resolved_at
		FROM reports
		WHERE id = $1 AND reporter_real_profile_id = $2
	`
	row := r.pool.QueryRow(ctx, sql, id, reporterID)
	return scanReport(row)
}

func (r *PostgresRepository) GetReportByID(ctx context.Context, id string) (*Report, error) {
	const sql = `
		SELECT id, reporter_real_profile_id, target_type, target_id, category, details, status, moderator_note, created_at, resolved_at
		FROM reports
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, sql, id)
	return scanReport(row)
}

func (r *PostgresRepository) ListReportsByTarget(ctx context.Context, targetType, targetID string) ([]*Report, error) {
	const sql = `
		SELECT id, reporter_real_profile_id, target_type, target_id, category, details, status, moderator_note, created_at, resolved_at
		FROM reports
		WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, sql, targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("list reports by target: %w", err)
	}
	defer rows.Close()

	var out []*Report
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reports by target: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) MarkReportsResolvedForCase(ctx context.Context, caseID string) error {
	const sql = `
		UPDATE reports
		SET status = 'resolved', resolved_at = now()
		WHERE id IN (
			SELECT report_id FROM case_reports WHERE case_id = $1
		)
	`
	if _, err := r.pool.Exec(ctx, sql, caseID); err != nil {
		return fmt.Errorf("mark reports resolved: %w", err)
	}
	return nil
}

func scanReport(row pgx.Row) (*Report, error) {
	var r Report
	if err := row.Scan(&r.ID, &r.ReporterRealProfileID, &r.TargetType, &r.TargetID, &r.Category, &r.Details, &r.Status, &r.ModeratorNote, &r.CreatedAt, &r.ResolvedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrReportNotFound
		}
		return nil, fmt.Errorf("scan report: %w", err)
	}
	return &r, nil
}

func (r *PostgresRepository) CreateCase(ctx context.Context, c *ModerationCase, reportIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertCaseSQL = `
		INSERT INTO moderation_cases (target_type, target_id, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	row := tx.QueryRow(ctx, insertCaseSQL, c.TargetType, c.TargetID, c.Status)
	if err := row.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return fmt.Errorf("insert case: %w", err)
	}

	const linkSQL = `
		INSERT INTO case_reports (case_id, report_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`
	for _, rid := range reportIDs {
		if _, err := tx.Exec(ctx, linkSQL, c.ID, rid); err != nil {
			return fmt.Errorf("link report to case: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit case: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetCase(ctx context.Context, id string) (*ModerationCase, error) {
	const sql = `
		SELECT id, target_type, target_id, status, outcome, notes, created_at, updated_at
		FROM moderation_cases
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, sql, id)
	return scanCase(row)
}

func (r *PostgresRepository) GetCaseReportIDs(ctx context.Context, id string) ([]string, error) {
	const sql = `SELECT report_id FROM case_reports WHERE case_id = $1 ORDER BY report_id`
	rows, err := r.pool.Query(ctx, sql, id)
	if err != nil {
		return nil, fmt.Errorf("list case report ids: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var rid string
		if err := rows.Scan(&rid); err != nil {
			return nil, fmt.Errorf("scan case report id: %w", err)
		}
		out = append(out, rid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case report ids: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) ListCases(ctx context.Context, status *string, cursor *time.Time, limit int) ([]*ModerationCase, *time.Time, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	const sql = `
		SELECT id, target_type, target_id, status, outcome, notes, created_at, updated_at
		FROM moderation_cases
		WHERE ($1::text IS NULL OR status = $1)
		  AND ($2::timestamptz IS NULL OR created_at < $2)
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, sql, status, cursor, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list cases: %w", err)
	}
	defer rows.Close()

	var out []*ModerationCase
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate cases: %w", err)
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

func (r *PostgresRepository) UpdateCase(ctx context.Context, c *ModerationCase) error {
	const sql = `
		UPDATE moderation_cases
		SET status = $2,
		    outcome = $3,
		    notes = $4,
		    updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	if err := r.pool.QueryRow(ctx, sql, c.ID, c.Status, c.Outcome, c.Notes).Scan(&c.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return ErrCaseNotFound
		}
		return fmt.Errorf("update case: %w", err)
	}
	return nil
}

func scanCase(row pgx.Row) (*ModerationCase, error) {
	var c ModerationCase
	if err := row.Scan(&c.ID, &c.TargetType, &c.TargetID, &c.Status, &c.Outcome, &c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCaseNotFound
		}
		return nil, fmt.Errorf("scan case: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) CreateAction(ctx context.Context, a *ModerationAction) error {
	const sql = `
		INSERT INTO moderation_actions (case_id, moderator_user_id, action_type, target_type, target_id, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	row := r.pool.QueryRow(ctx, sql, a.CaseID, a.ModeratorUserID, a.ActionType, a.TargetType, a.TargetID, a.Note)
	if err := row.Scan(&a.ID, &a.CreatedAt); err != nil {
		return fmt.Errorf("insert moderation action: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListActionsByCase(ctx context.Context, caseID string) ([]*ModerationAction, error) {
	const sql = `
		SELECT id, case_id, moderator_user_id, action_type, target_type, target_id, note, created_at
		FROM moderation_actions
		WHERE case_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, sql, caseID)
	if err != nil {
		return nil, fmt.Errorf("list actions by case: %w", err)
	}
	defer rows.Close()

	var out []*ModerationAction
	for rows.Next() {
		a, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate actions by case: %w", err)
	}
	return out, nil
}

func scanAction(row pgx.Row) (*ModerationAction, error) {
	var a ModerationAction
	if err := row.Scan(&a.ID, &a.CaseID, &a.ModeratorUserID, &a.ActionType, &a.TargetType, &a.TargetID, &a.Note, &a.CreatedAt); err != nil {
		return nil, fmt.Errorf("scan action: %w", err)
	}
	return &a, nil
}

func (r *PostgresRepository) UpdatePostModerationState(ctx context.Context, id, state string) error {
	const sql = `
		UPDATE posts
		SET moderation_state = $2, updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, state).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPostNotFound
		}
		return fmt.Errorf("update post moderation state: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateCommentModerationState(ctx context.Context, id, state string) error {
	const sql = `
		UPDATE comments
		SET moderation_state = $2, updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, state).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrCommentNotFound
		}
		return fmt.Errorf("update comment moderation state: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdatePersonaStatus(ctx context.Context, id, status string) error {
	const sql = `
		UPDATE personas
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, status).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrPersonaNotFound
		}
		return fmt.Errorf("update persona status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) UpdateUserStatus(ctx context.Context, id, status string) error {
	const sql = `
		UPDATE users
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id
	`
	var returnedID string
	if err := r.pool.QueryRow(ctx, sql, id, status).Scan(&returnedID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrReportTargetNotFound
		}
		return fmt.Errorf("update user status: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetPostByID(ctx context.Context, id string) (*postRef, error) {
	const sql = `SELECT id, persona_id FROM posts WHERE id = $1`
	row := r.pool.QueryRow(ctx, sql, id)
	var p postRef
	if err := row.Scan(&p.ID, &p.PersonaID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("select post: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) GetCommentByID(ctx context.Context, id string) (*commentRef, error) {
	const sql = `SELECT id, persona_id FROM comments WHERE id = $1`
	row := r.pool.QueryRow(ctx, sql, id)
	var c commentRef
	if err := row.Scan(&c.ID, &c.PersonaID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("select comment: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) GetPersonaByID(ctx context.Context, id string) (*personaRef, error) {
	const sql = `SELECT id, real_profile_id FROM personas WHERE id = $1`
	row := r.pool.QueryRow(ctx, sql, id)
	var p personaRef
	if err := row.Scan(&p.ID, &p.RealProfileID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPersonaNotFound
		}
		return nil, fmt.Errorf("select persona: %w", err)
	}
	return &p, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isCheckViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		return false
	}
	return pgErr.ConstraintName == constraint
}
