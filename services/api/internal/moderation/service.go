package moderation

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

// PersonaLookup provides read-only access to personas for validation.
type PersonaLookup interface {
	GetPersonaByID(ctx context.Context, id string) (*identity.Persona, error)
}

// Service implements the moderation domain logic.
type Service struct {
	cfg      *config.Config
	repo     Repository
	personas PersonaLookup
	limiter  auth.RateLimiter
}

// NewService creates a new moderation Service.
func NewService(cfg *config.Config, repo Repository, personas PersonaLookup, limiter auth.RateLimiter) *Service {
	return &Service{
		cfg:      cfg,
		repo:     repo,
		personas: personas,
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

// BlockPublic is the public view of a block.
type BlockPublic struct {
	ID        string
	Persona   *identity.Persona
	CreatedAt time.Time
}

// ReportPublic is the public view of a report submitted by the caller.
type ReportPublic struct {
	ID        string
	TargetType string
	TargetID  string
	Category  string
	Details   *string
	Status    string
	CreatedAt time.Time
	ResolvedAt *time.Time
}

// ModerationCasePublic is the staff-facing view of a case.
type ModerationCasePublic struct {
	ID        string
	TargetType string
	TargetID  string
	ReportIDs []string
	Status    string
	Outcome   *string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ModerationActionPublic is the staff-facing audit entry for a case.
type ModerationActionPublic struct {
	ID             string
	ModeratorUserID *string
	ActionType     string
	TargetType     *string
	TargetID       *string
	Note           *string
	CreatedAt      time.Time
}

// CreateBlockRequest contains the fields accepted when blocking a persona.
type CreateBlockRequest struct {
	BlockedPersonaID string
}

// CreateReportRequest contains the fields accepted when submitting a report.
type CreateReportRequest struct {
	TargetType string
	TargetID   string
	Category   string
	Details    *string
}

// CreateCaseRequest contains the fields accepted when creating a moderation case.
type CreateCaseRequest struct {
	TargetType string
	TargetID   string
	ReportIDs  []string
}

// UpdateCaseRequest contains the fields accepted when resolving a case.
type UpdateCaseRequest struct {
	Status  string
	Outcome *string
	Notes   *string
}

const (
	maxReportDetailsLen = 2000
	maxCaseNotesLen     = 4000
	reportRateLimit     = 50
	reportRateWindow    = time.Hour
	blockRateLimit      = 50
	blockRateWindow     = time.Hour
)

var (
	reportRate = auth.RateLimit{Count: reportRateLimit, Window: reportRateWindow}
	blockRate  = auth.RateLimit{Count: blockRateLimit, Window: blockRateWindow}
)

// CreateBlock directionally blocks a persona from the active persona.
func (s *Service) CreateBlock(ctx context.Context, blockerPersonaID string, req *CreateBlockRequest) (*BlockPublic, error) {
	if strings.TrimSpace(req.BlockedPersonaID) == "" {
		return nil, ErrBlockSelf
	}
	if req.BlockedPersonaID == blockerPersonaID {
		return nil, ErrBlockSelf
	}
	if _, err := s.personas.GetPersonaByID(ctx, req.BlockedPersonaID); err != nil {
		if errors.Is(err, identity.ErrPersonaNotFound) {
			return nil, ErrBlockSelf
		}
		return nil, fmt.Errorf("lookup blocked persona: %w", err)
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("block", blockerPersonaID), blockRate); err != nil || !allowed {
		return nil, &auth.RateLimitError{RetryAfter: retryAfter}
	}

	b := &Block{
		BlockerPersonaID: blockerPersonaID,
		BlockedPersonaID: req.BlockedPersonaID,
	}
	if err := s.repo.CreateBlock(ctx, b); err != nil {
		return nil, err
	}
	return s.toBlockPublic(ctx, b)
}

// DeleteBlock removes a block owned by the active persona.
func (s *Service) DeleteBlock(ctx context.Context, id, blockerPersonaID string) error {
	return s.repo.DeleteBlock(ctx, id, blockerPersonaID)
}

// ListBlocks returns the active persona's block list.
func (s *Service) ListBlocks(ctx context.Context, blockerPersonaID, cursor string, limit int) (*CursorPage[BlockPublic], error) {
	c, err := parseTimeCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	blocks, next, err := s.repo.ListBlocks(ctx, blockerPersonaID, c, limit)
	if err != nil {
		return nil, err
	}
	out := make([]BlockPublic, len(blocks))
	for i, b := range blocks {
		pub, err := s.toBlockPublic(ctx, b)
		if err != nil {
			return nil, err
		}
		out[i] = *pub
	}
	return toPage(out, next, limit), nil
}

// ListBlockedPersonaIDs returns the persona IDs blocked by the given persona.
func (s *Service) ListBlockedPersonaIDs(ctx context.Context, blockerPersonaID string) ([]string, error) {
	return s.repo.ListBlockedPersonaIDs(ctx, blockerPersonaID)
}

// IsBlocked returns true if either persona has blocked the other.
func (s *Service) IsBlocked(ctx context.Context, personaA, personaB string) (bool, error) {
	return s.repo.IsBlocked(ctx, personaA, personaB)
}

func (s *Service) toBlockPublic(ctx context.Context, b *Block) (*BlockPublic, error) {
	p, err := s.personas.GetPersonaByID(ctx, b.BlockedPersonaID)
	if err != nil {
		return nil, fmt.Errorf("load blocked persona: %w", err)
	}
	return &BlockPublic{
		ID:        b.ID,
		Persona:   p,
		CreatedAt: b.CreatedAt,
	}, nil
}

// CreateReport submits a report from the authenticated real profile.
func (s *Service) CreateReport(ctx context.Context, reporterID string, req *CreateReportRequest) (*ReportPublic, error) {
	if err := validateReportRequest(ctx, s, reporterID, req); err != nil {
		return nil, err
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("report", reporterID), reportRate); err != nil || !allowed {
		return nil, &auth.RateLimitError{RetryAfter: retryAfter}
	}

	r := &Report{
		ReporterRealProfileID: reporterID,
		TargetType:            req.TargetType,
		TargetID:              req.TargetID,
		Category:              req.Category,
		Details:               req.Details,
	}
	if err := s.repo.CreateReport(ctx, r); err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}
	return s.toReportPublic(r), nil
}

// GetReport returns a report submitted by the authenticated real profile.
func (s *Service) GetReport(ctx context.Context, reporterID, reportID string) (*ReportPublic, error) {
	r, err := s.repo.GetReportByIDAndReporter(ctx, reportID, reporterID)
	if err != nil {
		return nil, err
	}
	return s.toReportPublic(r), nil
}

func validateReportRequest(ctx context.Context, s *Service, reporterID string, req *CreateReportRequest) error {
	validCategories := map[string]bool{
		"harassment": true, "hateSpeech": true, "harmfulContent": true, "spam": true,
		"sexualContent": true, "doxxing": true, "impersonation": true, "illegalContent": true, "other": true,
	}
	if !validCategories[req.Category] {
		return ErrReportInvalidTarget
	}
	if req.Details != nil && len(*req.Details) > maxReportDetailsLen {
		return ErrReportInvalidTarget
	}
	switch req.TargetType {
	case "post":
		p, err := s.repo.GetPostByID(ctx, req.TargetID)
		if err != nil {
			if err == ErrPostNotFound {
				return ErrReportTargetNotFound
			}
			return fmt.Errorf("validate report target post: %w", err)
		}
		if p.PersonaID == reporterID {
			return ErrReportSelf
		}
	case "comment":
		c, err := s.repo.GetCommentByID(ctx, req.TargetID)
		if err != nil {
			if err == ErrCommentNotFound {
				return ErrReportTargetNotFound
			}
			return fmt.Errorf("validate report target comment: %w", err)
		}
		if c.PersonaID == reporterID {
			return ErrReportSelf
		}
	case "persona":
		p, err := s.repo.GetPersonaByID(ctx, req.TargetID)
		if err != nil {
			if err == ErrPersonaNotFound {
				return ErrReportTargetNotFound
			}
			return fmt.Errorf("validate report target persona: %w", err)
		}
		if p.RealProfileID == reporterID {
			return ErrReportSelf
		}
	default:
		return ErrReportInvalidTarget
	}
	return nil
}

func (s *Service) toReportPublic(r *Report) *ReportPublic {
	return &ReportPublic{
		ID:         r.ID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Category:   r.Category,
		Details:    r.Details,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt,
		ResolvedAt: r.ResolvedAt,
	}
}

// CreateCase creates a moderation case from one or more reports.
func (s *Service) CreateCase(ctx context.Context, moderatorID string, req *CreateCaseRequest) (*ModerationCasePublic, error) {
	if len(req.ReportIDs) == 0 {
		return nil, ErrCaseInvalidOutcome
	}
	validTargetTypes := map[string]bool{"post": true, "comment": true, "persona": true}
	if !validTargetTypes[req.TargetType] {
		return nil, ErrCaseInvalidOutcome
	}
	// Verify all reports exist and match the target.
	for _, rid := range req.ReportIDs {
		r, err := s.repo.GetReportByID(ctx, rid)
		if err != nil {
			if err == ErrReportNotFound {
				return nil, ErrReportNotFound
			}
			return nil, fmt.Errorf("lookup report: %w", err)
		}
		if r.TargetType != req.TargetType || r.TargetID != req.TargetID {
			return nil, ErrReportInvalidTarget
		}
	}

	c := &ModerationCase{
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Status:     "open",
	}
	if err := s.repo.CreateCase(ctx, c, req.ReportIDs); err != nil {
		return nil, fmt.Errorf("create case: %w", err)
	}

	action := &ModerationAction{
		CaseID:          &c.ID,
		ModeratorUserID: &moderatorID,
		ActionType:      "open",
		TargetType:      &req.TargetType,
		TargetID:        &req.TargetID,
	}
	_ = s.repo.CreateAction(ctx, action)

	return s.toCasePublic(ctx, c)
}

// GetCase returns a moderation case by ID.
func (s *Service) GetCase(ctx context.Context, id string) (*ModerationCasePublic, error) {
	c, err := s.repo.GetCase(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toCasePublic(ctx, c)
}

// ListCases returns moderation cases, optionally filtered by status.
func (s *Service) ListCases(ctx context.Context, status, cursor string, limit int) (*CursorPage[ModerationCasePublic], error) {
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	c, err := parseTimeCursor(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}
	cases, next, err := s.repo.ListCases(ctx, statusPtr, c, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ModerationCasePublic, len(cases))
	for i, c := range cases {
		pub, err := s.toCasePublic(ctx, c)
		if err != nil {
			return nil, err
		}
		out[i] = *pub
	}
	return toPage(out, next, limit), nil
}

// UpdateCase resolves or transitions a moderation case and applies the outcome.
func (s *Service) UpdateCase(ctx context.Context, moderatorID, caseID string, req *UpdateCaseRequest) (*ModerationCasePublic, error) {
	c, err := s.repo.GetCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if c.Status == "resolved" {
		return nil, ErrCaseInvalidStatus
	}

	validStatuses := map[string]bool{"underReview": true, "resolved": true}
	if !validStatuses[req.Status] {
		return nil, ErrCaseInvalidStatus
	}
	c.Status = req.Status

	if req.Status == "resolved" {
		if req.Outcome == nil {
			return nil, ErrCaseInvalidOutcome
		}
		if err := s.applyOutcome(ctx, moderatorID, c, *req.Outcome, req.Notes); err != nil {
			return nil, err
		}
		c.Outcome = req.Outcome
		if req.Notes != nil && len(*req.Notes) > maxCaseNotesLen {
			return nil, ErrCaseInvalidOutcome
		}
		c.Notes = req.Notes
		if err := s.repo.MarkReportsResolvedForCase(ctx, c.ID); err != nil {
			return nil, fmt.Errorf("resolve reports: %w", err)
		}
	} else {
		// underReview: just record the transition.
		if req.Notes != nil && len(*req.Notes) > maxCaseNotesLen {
			return nil, ErrCaseInvalidOutcome
		}
		c.Notes = req.Notes
	}

	if err := s.repo.UpdateCase(ctx, c); err != nil {
		return nil, err
	}

	actionType := req.Status
	if req.Status == "resolved" && req.Outcome != nil {
		actionType = *req.Outcome
	}
	action := &ModerationAction{
		CaseID:          &c.ID,
		ModeratorUserID: &moderatorID,
		ActionType:      actionType,
		TargetType:      &c.TargetType,
		TargetID:        &c.TargetID,
		Note:            req.Notes,
	}
	_ = s.repo.CreateAction(ctx, action)

	return s.toCasePublic(ctx, c)
}

// applyOutcome performs the content/account mutation for a resolved case.
func (s *Service) applyOutcome(ctx context.Context, moderatorID string, c *ModerationCase, outcome string, notes *string) error {
	validOutcomes := map[string]bool{
		"noAction": true, "warn": true, "hide": true, "remove": true, "restore": true,
		"restrictPersona": true, "suspendAccount": true, "banAccount": true,
	}
	if !validOutcomes[outcome] {
		return ErrCaseInvalidOutcome
	}

	switch outcome {
	case "hide":
		switch c.TargetType {
		case "post":
			return s.repo.UpdatePostModerationState(ctx, c.TargetID, "hidden")
		case "comment":
			return s.repo.UpdateCommentModerationState(ctx, c.TargetID, "hidden")
		}
	case "remove":
		switch c.TargetType {
		case "post":
			return s.repo.UpdatePostModerationState(ctx, c.TargetID, "deleted")
		case "comment":
			return s.repo.UpdateCommentModerationState(ctx, c.TargetID, "deleted")
		}
	case "restore":
		switch c.TargetType {
		case "post":
			return s.repo.UpdatePostModerationState(ctx, c.TargetID, "published")
		case "comment":
			return s.repo.UpdateCommentModerationState(ctx, c.TargetID, "published")
		}
	case "restrictPersona":
		if c.TargetType == "persona" {
			return s.repo.UpdatePersonaStatus(ctx, c.TargetID, "restricted")
		}
		// For post/comment, restrict the authoring persona.
		return s.restrictAuthorPersona(ctx, c.TargetType, c.TargetID)
	case "suspendAccount":
		return s.suspendOrBanAuthorAccount(ctx, c.TargetType, c.TargetID, "suspended")
	case "banAccount":
		return s.suspendOrBanAuthorAccount(ctx, c.TargetType, c.TargetID, "banned")
	case "warn":
		// No automatic state change; note is stored on the case.
	}
	return nil
}

func (s *Service) restrictAuthorPersona(ctx context.Context, targetType, targetID string) error {
	switch targetType {
	case "post":
		p, err := s.repo.GetPostByID(ctx, targetID)
		if err != nil {
			return err
		}
		return s.repo.UpdatePersonaStatus(ctx, p.PersonaID, "restricted")
	case "comment":
		c, err := s.repo.GetCommentByID(ctx, targetID)
		if err != nil {
			return err
		}
		return s.repo.UpdatePersonaStatus(ctx, c.PersonaID, "restricted")
	case "persona":
		return s.repo.UpdatePersonaStatus(ctx, targetID, "restricted")
	}
	return nil
}

func (s *Service) suspendOrBanAuthorAccount(ctx context.Context, targetType, targetID, status string) error {
	switch targetType {
	case "post":
		p, err := s.repo.GetPostByID(ctx, targetID)
		if err != nil {
			return err
		}
		persona, err := s.repo.GetPersonaByID(ctx, p.PersonaID)
		if err != nil {
			return err
		}
		return s.repo.UpdateUserStatus(ctx, persona.RealProfileID, status)
	case "comment":
		c, err := s.repo.GetCommentByID(ctx, targetID)
		if err != nil {
			return err
		}
		persona, err := s.repo.GetPersonaByID(ctx, c.PersonaID)
		if err != nil {
			return err
		}
		return s.repo.UpdateUserStatus(ctx, persona.RealProfileID, status)
	case "persona":
		persona, err := s.repo.GetPersonaByID(ctx, targetID)
		if err != nil {
			return err
		}
		return s.repo.UpdateUserStatus(ctx, persona.RealProfileID, status)
	}
	return nil
}

// ListCaseActions returns the audit history for a case.
func (s *Service) ListCaseActions(ctx context.Context, caseID string) ([]ModerationActionPublic, error) {
	actions, err := s.repo.ListActionsByCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	out := make([]ModerationActionPublic, len(actions))
	for i, a := range actions {
		out[i] = ModerationActionPublic{
			ID:             a.ID,
			ModeratorUserID: a.ModeratorUserID,
			ActionType:     a.ActionType,
			TargetType:     a.TargetType,
			TargetID:       a.TargetID,
			Note:           a.Note,
			CreatedAt:      a.CreatedAt,
		}
	}
	return out, nil
}

func (s *Service) toCasePublic(ctx context.Context, c *ModerationCase) (*ModerationCasePublic, error) {
	reportIDs, err := s.repo.GetCaseReportIDs(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return &ModerationCasePublic{
		ID:         c.ID,
		TargetType: c.TargetType,
		TargetID:   c.TargetID,
		ReportIDs:  reportIDs,
		Status:     c.Status,
		Outcome:    c.Outcome,
		Notes:      c.Notes,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}, nil
}

func toPage[T any](data []T, next *time.Time, limit int) *CursorPage[T] {
	var nextCursor *string
	if next != nil {
		cs := encodeTimeCursor(*next)
		nextCursor = &cs
	}
	return &CursorPage[T]{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    next != nil,
		Limit:      limit,
	}
}

func rateLimitKey(prefix, id string) string {
	return fmt.Sprintf("moderation:%s:%s", prefix, id)
}

func parseTimeCursor(s string) (*time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func encodeTimeCursor(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

// Errors alias for external packages.
var (
	ErrorsIs = errors.Is
)
