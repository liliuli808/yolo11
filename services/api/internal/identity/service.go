package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/platform/config"
)

// ContentService is the subset of the content service needed by identity.
type ContentService interface {
	ListPersonaPostsForIdentity(ctx context.Context, personaID, cursor string, limit int, viewerPersonaID *string) (*CursorPagePost, error)
}

// Service implements the identity domain logic.
type Service struct {
	cfg            *config.Config
	repo           Repository
	authRepo       auth.Repository
	mailer         auth.Mailer
	limiter        auth.RateLimiter
	contentService ContentService
	codeKey        []byte
}

// NewService creates a new identity Service.
func NewService(cfg *config.Config, repo Repository, authRepo auth.Repository, mailer auth.Mailer, limiter auth.RateLimiter, contentService ContentService) *Service {
	return &Service{
		cfg:            cfg,
		repo:           repo,
		authRepo:       authRepo,
		mailer:         mailer,
		limiter:        limiter,
		contentService: contentService,
		codeKey:        []byte(cfg.EmailCodeKey),
	}
}

// PrivatePersona is the owner view of a persona.
type PrivatePersona struct {
	ID          string
	Alias       string
	Bio         *string
	AvatarSeed  string
	AvatarColor string
	Status      string
	IsDefault   bool
	NoteCount   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PublicPersona is the public view of a persona; it never includes real-profile fields.
type PublicPersona struct {
	ID          string
	Alias       string
	Bio         *string
	AvatarSeed  string
	AvatarColor string
	NoteCount   int
	CreatedAt   time.Time
	IsBlocked   bool
}

// CursorPagePost is an empty placeholder for the public persona posts endpoint.
type CursorPagePost struct {
	Data       []any
	NextCursor *string
	HasMore    bool
	Limit      int
}

const (
	maxPersonasPerAccount = 5
	exportCooldown        = 30 * 24 * time.Hour
	defaultAvatarColor    = "#6366F1"
)

// Rate limits for email-change requests. Applied independently by scope.
var (
	emailChangeByEmail = auth.RateLimit{Count: 5, Window: 10 * time.Minute}
	emailChangeByIP    = auth.RateLimit{Count: 20, Window: 10 * time.Minute}
	emailChangeByFP    = auth.RateLimit{Count: 10, Window: 10 * time.Minute}
)

// GetMe returns the authenticated real profile.
func (s *Service) GetMe(ctx context.Context, userID string) (*RealProfile, error) {
	profile, err := s.repo.GetRealProfileByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// RequestEmailChange sends a verification code to the proposed new email address.
func (s *Service) RequestEmailChange(ctx context.Context, userID, newEmail, ip, fingerprint string) error {
	if err := validateEmail(newEmail); err != nil {
		return ErrInvalidEmail
	}
	normalized := NormalizeEmail(newEmail)

	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("email_change:request", "email", normalized), emailChangeByEmail); err != nil || !allowed {
		return &auth.RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("email_change:request", "ip", ip), emailChangeByIP); err != nil || !allowed {
		return &auth.RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, rateLimitKey("email_change:request", "fp", fingerprint), emailChangeByFP); err != nil || !allowed {
		return &auth.RateLimitError{RetryAfter: retryAfter}
	}

	profile, err := s.repo.GetRealProfileByID(ctx, userID)
	if err != nil {
		return err
	}
	if profile.EmailNormalized == normalized {
		return ErrInvalidEmail
	}

	used, err := s.repo.EmailExistsForOther(ctx, userID, normalized)
	if err != nil {
		return fmt.Errorf("check email availability: %w", err)
	}
	if used {
		return ErrEmailAlreadyUsed
	}

	code, err := generateEmailCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	hash := hashEmailCode(s.codeKey, code)
	expiresAt := time.Now().UTC().Add(10 * time.Minute)

	if err := s.authRepo.CreateEmailCode(ctx, &auth.EmailCode{
		UserID:      &userID,
		Email:       normalized,
		CodeHash:    hash,
		Purpose:     "email_change",
		ExpiresAt:   expiresAt,
		MaxAttempts: 5,
	}); err != nil {
		return fmt.Errorf("store email change code: %w", err)
	}

	if err := s.mailer.SendEmailCode(ctx, normalized, code, 10*time.Minute); err != nil {
		return fmt.Errorf("send email change code: %w", err)
	}

	return nil
}

// ConfirmEmailChange verifies the code and updates the real profile email.
func (s *Service) ConfirmEmailChange(ctx context.Context, userID, newEmail, code string) (*RealProfile, error) {
	if err := validateEmail(newEmail); err != nil {
		return nil, ErrInvalidEmail
	}
	normalized := NormalizeEmail(newEmail)

	profile, err := s.repo.GetRealProfileByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile.EmailNormalized == normalized {
		return nil, ErrInvalidEmail
	}

	if err := s.verifyEmailCode(ctx, normalized, code, "email_change"); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateEmail(ctx, userID, normalized); err != nil {
		return nil, fmt.Errorf("update email: %w", err)
	}

	return s.repo.GetRealProfileByID(ctx, userID)
}

// RequestDataExport creates a new pending data export request.
func (s *Service) RequestDataExport(ctx context.Context, userID, format string) (*DataExport, error) {
	if format != "json" && format != "zip" {
		return nil, ErrInvalidExportFormat
	}

	latest, err := s.repo.GetLatestDataExport(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("lookup latest export: %w", err)
	}
	if latest != nil && time.Since(latest.RequestedAt) < exportCooldown {
		return nil, ErrExportRateLimited
	}

	export := &DataExport{
		RealProfileID: userID,
		Status:        "pending",
		Format:        format,
	}
	if err := s.repo.CreateDataExport(ctx, export); err != nil {
		return nil, fmt.Errorf("create export: %w", err)
	}
	return export, nil
}

// GetDataExport returns a single export request owned by the real profile.
func (s *Service) GetDataExport(ctx context.Context, userID, exportID string) (*DataExport, error) {
	return s.repo.GetDataExportByID(ctx, exportID, userID)
}

// ListPersonas returns all personas owned by the real profile.
func (s *Service) ListPersonas(ctx context.Context, userID string) ([]*PrivatePersona, error) {
	personas, err := s.repo.ListPersonasByRealProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list personas: %w", err)
	}
	out := make([]*PrivatePersona, len(personas))
	for i, p := range personas {
		out[i] = toPrivatePersona(p)
	}
	return out, nil
}

// CreatePersona creates a new active persona under the real profile.
func (s *Service) CreatePersona(ctx context.Context, userID string, req *PersonaCreateRequest) (*PrivatePersona, error) {
	if err := validateAlias(req.Alias); err != nil {
		return nil, err
	}
	if req.Bio != nil && len(*req.Bio) > 160 {
		return nil, ErrPersonaAliasDisallowed // generic validation error
	}
	avatarSeed, avatarColor := defaultAvatar(req)

	count, err := s.repo.CountActivePersonas(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count personas: %w", err)
	}
	if count >= maxPersonasPerAccount {
		return nil, ErrPersonaMaxReached
	}

	p := &Persona{
		RealProfileID: userID,
		Alias:         strings.TrimSpace(req.Alias),
		Bio:           req.Bio,
		AvatarSeed:    avatarSeed,
		AvatarColor:   avatarColor,
		Status:        "active",
		IsDefault:     false,
		NoteCount:     0,
	}
	if err := s.repo.CreatePersona(ctx, p); err != nil {
		return nil, err
	}

	// The first active persona becomes the default automatically.
	profile, err := s.repo.GetRealProfileByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("reload profile: %w", err)
	}
	if profile.DefaultPersonaID == nil {
		if err := s.repo.SetDefaultPersona(ctx, p.ID, userID); err != nil {
			return nil, fmt.Errorf("set default persona: %w", err)
		}
		p.IsDefault = true
	}

	return toPrivatePersona(p), nil
}

// GetPrivatePersona returns a single owned persona.
func (s *Service) GetPrivatePersona(ctx context.Context, userID, personaID string) (*PrivatePersona, error) {
	p, err := s.repo.GetPersonaByIDAndRealProfile(ctx, personaID, userID)
	if err != nil {
		return nil, err
	}
	return toPrivatePersona(p), nil
}

// UpdatePersona updates an owned active persona.
func (s *Service) UpdatePersona(ctx context.Context, userID, personaID string, req *PersonaUpdateRequest) (*PrivatePersona, error) {
	p, err := s.repo.GetPersonaByIDAndRealProfile(ctx, personaID, userID)
	if err != nil {
		return nil, err
	}
	if p.Status == "archived" {
		return nil, ErrPersonaArchived
	}
	if req.Alias != nil {
		if err := validateAlias(*req.Alias); err != nil {
			return nil, err
		}
		p.Alias = strings.TrimSpace(*req.Alias)
	}
	if req.Bio != nil && len(*req.Bio) > 160 {
		return nil, ErrPersonaAliasDisallowed
	}
	if req.Bio != nil {
		p.Bio = req.Bio
	}
	if req.AvatarSeed != nil {
		p.AvatarSeed = *req.AvatarSeed
	}
	if req.AvatarColor != nil {
		if !isHexColor(*req.AvatarColor) {
			return nil, ErrPersonaAliasDisallowed
		}
		p.AvatarColor = *req.AvatarColor
	}

	if err := s.repo.UpdatePersona(ctx, p); err != nil {
		return nil, err
	}

	// Reload to capture updated_at and any default changes.
	p, err = s.repo.GetPersonaByIDAndRealProfile(ctx, personaID, userID)
	if err != nil {
		return nil, fmt.Errorf("reload persona: %w", err)
	}
	return toPrivatePersona(p), nil
}

// ArchivePersona archives an owned persona.
func (s *Service) ArchivePersona(ctx context.Context, userID, personaID string) error {
	p, err := s.repo.GetPersonaByIDAndRealProfile(ctx, personaID, userID)
	if err != nil {
		return err
	}
	if p.Status == "archived" {
		return ErrPersonaArchived
	}
	wasDefault := p.IsDefault

	if err := s.repo.ArchivePersona(ctx, personaID, userID); err != nil {
		return err
	}

	if wasDefault {
		if err := s.setNextDefaultPersona(ctx, userID); err != nil {
			return fmt.Errorf("set next default persona: %w", err)
		}
	}
	return nil
}

// SetDefaultPersona marks an active owned persona as the default.
func (s *Service) SetDefaultPersona(ctx context.Context, userID, personaID string) (*PrivatePersona, error) {
	p, err := s.repo.GetPersonaByIDAndRealProfile(ctx, personaID, userID)
	if err != nil {
		return nil, err
	}
	if p.Status != "active" {
		return nil, ErrPersonaRestricted
	}
	if err := s.repo.SetDefaultPersona(ctx, personaID, userID); err != nil {
		return nil, err
	}
	p.IsDefault = true
	return toPrivatePersona(p), nil
}

// GetPublicPersona returns the public view of an active persona.
func (s *Service) GetPublicPersona(ctx context.Context, personaID string) (*PublicPersona, error) {
	p, err := s.repo.GetPersonaByID(ctx, personaID)
	if err != nil {
		return nil, err
	}
	if p.Status != "active" {
		return nil, ErrPersonaNotFound
	}
	return &PublicPersona{
		ID:          p.ID,
		Alias:       p.Alias,
		Bio:         p.Bio,
		AvatarSeed:  p.AvatarSeed,
		AvatarColor: p.AvatarColor,
		NoteCount:   p.NoteCount,
		CreatedAt:   p.CreatedAt,
		IsBlocked:   false,
	}, nil
}

// ListPersonaPosts delegates to the content service for a persona's public posts.
func (s *Service) ListPersonaPosts(ctx context.Context, personaID, cursor string, limit int) (*CursorPagePost, error) {
	p, err := s.repo.GetPersonaByID(ctx, personaID)
	if err != nil {
		return nil, err
	}
	if p.Status != "active" {
		return nil, ErrPersonaNotFound
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if s.contentService == nil {
		return &CursorPagePost{Data: []any{}, Limit: limit}, nil
	}
	return s.contentService.ListPersonaPostsForIdentity(ctx, personaID, cursor, limit, nil)
}

// ProcessPendingExports finds pending exports older than a short grace period
// and transitions them to ready with a placeholder download URL. It is a
// minimal background-ready stub; callers are responsible for scheduling.
func (s *Service) ProcessPendingExports(ctx context.Context) (int, error) {
	const gracePeriod = 5 * time.Minute
	const downloadTTL = 7 * 24 * time.Hour

	exports, err := s.repo.ListPendingDataExports(ctx, time.Now().UTC().Add(-gracePeriod))
	if err != nil {
		return 0, err
	}

	processed := 0
	expiresAt := time.Now().UTC().Add(downloadTTL)
	for _, e := range exports {
		url := fmt.Sprintf("https://example.com/exports/%s", e.ID)
		if err := s.repo.UpdateDataExportReady(ctx, e.ID, url, expiresAt); err != nil {
			return processed, fmt.Errorf("mark export %s ready: %w", e.ID, err)
		}
		processed++
	}
	return processed, nil
}

// CleanupOnAccountDeletion archives all personas when an account enters deletion.
func (s *Service) CleanupOnAccountDeletion(ctx context.Context, userID string) error {
	return s.repo.ArchiveAllPersonasForRealProfile(ctx, userID)
}

func (s *Service) setNextDefaultPersona(ctx context.Context, userID string) error {
	personas, err := s.repo.ListPersonasByRealProfile(ctx, userID)
	if err != nil {
		return err
	}
	for _, p := range personas {
		if p.Status == "active" {
			return s.repo.SetDefaultPersona(ctx, p.ID, userID)
		}
	}
	return s.repo.ClearDefaultPersonaForRealProfile(ctx, userID)
}

func toPrivatePersona(p *Persona) *PrivatePersona {
	return &PrivatePersona{
		ID:          p.ID,
		Alias:       p.Alias,
		Bio:         p.Bio,
		AvatarSeed:  p.AvatarSeed,
		AvatarColor: p.AvatarColor,
		Status:      p.Status,
		IsDefault:   p.IsDefault,
		NoteCount:   p.NoteCount,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func defaultAvatar(req *PersonaCreateRequest) (string, string) {
	seed := req.Alias
	if strings.TrimSpace(req.AvatarSeed) != "" {
		seed = strings.TrimSpace(req.AvatarSeed)
	}
	color := defaultAvatarColor
	if isHexColor(req.AvatarColor) {
		color = req.AvatarColor
	}
	return seed, color
}

func validateAlias(alias string) error {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" || trimmed != alias || len(alias) > 32 {
		return ErrPersonaAliasDisallowed
	}
	return nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 254 {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}
	if NormalizeEmail(addr.Address) != NormalizeEmail(email) {
		return ErrInvalidEmail
	}
	return nil
}

// NormalizeEmail lowercases and trims whitespace from an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func isHexColor(s string) bool {
	return hexColorRe.MatchString(s)
}

func (s *Service) verifyEmailCode(ctx context.Context, email, code, purpose string) error {
	active, err := s.authRepo.GetActiveEmailCode(ctx, email, purpose)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCode, err)
	}
	if active == nil {
		return ErrInvalidCode
	}

	expected := hashEmailCode(s.codeKey, code)
	if !hmacEqual(active.CodeHash, expected) {
		_ = s.authRepo.IncrementEmailCodeAttempts(ctx, active.ID)
		return ErrInvalidCode
	}

	if err := s.authRepo.MarkEmailCodeUsed(ctx, active.ID); err != nil {
		return ErrInvalidCode
	}
	return nil
}

func generateEmailCode() (string, error) {
	const max = 1_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashEmailCode(key []byte, code string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func hmacEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func rateLimitKey(prefix, scope, value string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, scope, value)
}

// MaskEmail returns a masked representation of an email address.
func MaskEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "•••"
	}
	local, domain := email[:at], email[at+1:]
	if len(local) == 0 {
		return "•••@" + domain
	}
	return string(local[0]) + "•••@" + domain
}
