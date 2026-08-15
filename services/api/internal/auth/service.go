package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yiguan/api/internal/platform/config"
)

// Common auth domain errors.
var (
	ErrInvalidEmail           = errors.New("invalid email")
	ErrInvalidCode            = errors.New("invalid or expired verification code")
	ErrRateLimited            = errors.New("rate limited")
	ErrSessionExpired         = errors.New("session expired")
	ErrSessionRevoked         = errors.New("session revoked")
	ErrInvalidToken           = errors.New("invalid token")
	ErrUserNotFound           = errors.New("user not found")
	ErrDeletionAlreadyPending = errors.New("account deletion already pending")
	ErrDeletionInvalidCode    = errors.New("invalid deletion verification code")
)

// RateLimitError carries the retry window for a rate-limited request.
type RateLimitError struct {
	RetryAfter time.Duration
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	return "rate limited"
}

// TokenResponse is returned when a session is first created.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	PersonaID    *string
	IsStaff      bool
}

// AccessTokenResponse is returned when an access token is refreshed.
type AccessTokenResponse struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int
	PersonaID   *string
	IsStaff     bool
}

// RefreshResult contains both a new access token and a rotated refresh token.
type RefreshResult struct {
	AccessTokenResponse
	RefreshToken string
}

// AccountDeletionState represents the confirmed deletion request.
type AccountDeletionState struct {
	GracePeriodEndsAt time.Time
}

// AccessTokenClaims are extracted from a validated access token.
type AccessTokenClaims struct {
	UserID    string
	SessionID string
	IsStaff   bool
}

type accessClaims struct {
	UserID    string `json:"sub"`
	SessionID string `json:"sid"`
	Type      string `json:"type"`
	IsStaff   bool   `json:"is_staff"`
	jwt.RegisteredClaims
}

// Service implements the auth domain logic.
type Service struct {
	cfg     *config.Config
	repo    Repository
	mailer  Mailer
	limiter RateLimiter
	signer  []byte
	codeKey []byte
	keys    keyBuilder

	// IdentityCleanup is an optional hook invoked when an account enters the
	// deletion grace period. It is used by the identity subsystem to archive
	// personas and clear default persona references.
	IdentityCleanup func(ctx context.Context, userID string) error
}

// NewService creates a new auth Service.
func NewService(cfg *config.Config, repo Repository, mailer Mailer, limiter RateLimiter) *Service {
	return &Service{
		cfg:     cfg,
		repo:    repo,
		mailer:  mailer,
		limiter: limiter,
		signer:  []byte(cfg.JWTSigningKey),
		codeKey: []byte(cfg.EmailCodeKey),
	}
}

// Predefined rate limits. These apply independently by scope.
var (
	codeRequestByEmail = RateLimit{Count: 5, Window: 10 * time.Minute}
	codeRequestByIP    = RateLimit{Count: 20, Window: 10 * time.Minute}
	codeRequestByFP    = RateLimit{Count: 10, Window: 10 * time.Minute}

	codeVerifyByEmail = RateLimit{Count: 10, Window: 10 * time.Minute}
	codeVerifyByIP    = RateLimit{Count: 40, Window: 10 * time.Minute}
	codeVerifyByFP    = RateLimit{Count: 20, Window: 10 * time.Minute}
)

// RequestEmailCode generates a six-digit code, hashes it with a keyed hash,
// stores it, and sends it to the supplied email address.
func (s *Service) RequestEmailCode(ctx context.Context, email, purpose, ip, fingerprint string) error {
	if err := validateEmail(email); err != nil {
		return ErrInvalidEmail
	}
	normalized := NormalizeEmail(email)

	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.email("code:request", normalized), codeRequestByEmail); err != nil || !allowed {
		return &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.ip("code:request", ip), codeRequestByIP); err != nil || !allowed {
		return &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.fingerprint("code:request", fingerprint), codeRequestByFP); err != nil || !allowed {
		return &RateLimitError{RetryAfter: retryAfter}
	}

	user, err := s.repo.FindOrCreateUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}

	code, err := generateEmailCode()
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	hash := hashEmailCode(s.codeKey, code)
	expiresAt := time.Now().UTC().Add(10 * time.Minute)

	if err := s.repo.CreateEmailCode(ctx, &EmailCode{
		UserID:      &user.ID,
		Email:       normalized,
		CodeHash:    hash,
		Purpose:     purpose,
		ExpiresAt:   expiresAt,
		MaxAttempts: 5,
		IPAddress:   ip,
		Fingerprint: fingerprint,
	}); err != nil {
		return fmt.Errorf("store code: %w", err)
	}

	if err := s.mailer.SendEmailCode(ctx, normalized, code, 10*time.Minute); err != nil {
		return fmt.Errorf("send code: %w", err)
	}

	s.audit(ctx, &user.ID, nil, "email_code.requested", ip, "", fingerprint, map[string]any{"purpose": purpose})
	return nil
}

// CreateEmailSession verifies a code and creates a session with access and
// refresh tokens.
func (s *Service) CreateEmailSession(ctx context.Context, email, code, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	if err := validateEmail(email); err != nil {
		return nil, ErrInvalidEmail
	}
	normalized := NormalizeEmail(email)

	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.email("code:verify", normalized), codeVerifyByEmail); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.ip("code:verify", ip), codeVerifyByIP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.fingerprint("code:verify", fingerprint), codeVerifyByFP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	user, err := s.repo.FindOrCreateUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	if err := s.VerifyEmailCode(ctx, normalized, code, "login"); err != nil {
		return nil, err
	}

	refreshToken, refreshHash, err := s.newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	session := &Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        now.Add(s.cfg.RefreshTokenTTL),
		IPAddress:        ip,
		UserAgent:        userAgent,
		Fingerprint:      fingerprint,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	accessToken, expiresIn, err := s.issueAccessToken(user.ID, session.ID, s.isStaff(user.EmailNormalized))
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	s.audit(ctx, &user.ID, &session.ID, "session.created", ip, userAgent, fingerprint, map[string]any{"is_staff": s.isStaff(user.EmailNormalized)})

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		IsStaff:      s.isStaff(user.EmailNormalized),
	}, nil
}

// VerifyEmailCode checks an active code for the given email and purpose and
// marks it used when valid.
func (s *Service) VerifyEmailCode(ctx context.Context, email, code, purpose string) error {
	active, err := s.repo.GetActiveEmailCode(ctx, email, purpose)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCode, err)
	}
	if active == nil {
		return ErrInvalidCode
	}

	expected := hashEmailCode(s.codeKey, code)
	if !hmacEqual(active.CodeHash, expected) {
		_ = s.repo.IncrementEmailCodeAttempts(ctx, active.ID)
		return ErrInvalidCode
	}

	if err := s.repo.MarkEmailCodeUsed(ctx, active.ID); err != nil {
		return ErrInvalidCode
	}
	return nil
}

// RefreshSession rotates the refresh token and issues a new access token.
func (s *Service) RefreshSession(ctx context.Context, refreshToken, ip, fingerprint, userAgent string) (*RefreshResult, error) {
	if refreshToken == "" {
		return nil, ErrSessionExpired
	}

	hash := hashRefreshToken(refreshToken)
	session, err := s.repo.GetSessionByRefreshHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("lookup session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionExpired
	}

	newRefreshToken, newHash, err := s.newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	newSession := &Session{
		UserID:           session.UserID,
		RefreshTokenHash: newHash,
		ExpiresAt:        now.Add(s.cfg.RefreshTokenTTL),
		IPAddress:        ip,
		UserAgent:        userAgent,
		Fingerprint:      fingerprint,
	}
	if err := s.repo.CreateSession(ctx, newSession); err != nil {
		return nil, fmt.Errorf("create rotated session: %w", err)
	}

	if err := s.repo.RevokeSession(ctx, session.ID); err != nil {
		return nil, fmt.Errorf("revoke old session: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	accessToken, expiresIn, err := s.issueAccessToken(session.UserID, newSession.ID, s.isStaff(user.EmailNormalized))
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	s.audit(ctx, &session.UserID, &newSession.ID, "session.rotated", ip, userAgent, fingerprint, nil)

	return &RefreshResult{
		AccessTokenResponse: AccessTokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			IsStaff:     s.isStaff(user.EmailNormalized),
		},
		RefreshToken: newRefreshToken,
	}, nil
}

// Logout revokes the given session.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if err := s.repo.RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	s.audit(ctx, nil, &sessionID, "session.revoked", "", "", "", nil)
	return nil
}

// RevokeAllSessions revokes every active session for a user.
func (s *Service) RevokeAllSessions(ctx context.Context, userID string) error {
	if err := s.repo.RevokeAllSessionsForUser(ctx, userID); err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}
	s.audit(ctx, &userID, nil, "sessions.revoked_all", "", "", "", nil)
	return nil
}

// RequestAccountDeletion marks a user for deletion and revokes all sessions.
func (s *Service) RequestAccountDeletion(ctx context.Context, userID string) (*AccountDeletionState, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserNotFound, err)
	}
	if user.Status == "deleting" {
		return nil, ErrDeletionAlreadyPending
	}

	const gracePeriod = 30 * 24 * time.Hour
	if err := s.repo.MarkUserDeleting(ctx, userID, gracePeriod); err != nil {
		return nil, fmt.Errorf("mark deleting: %w", err)
	}
	if s.IdentityCleanup != nil {
		if err := s.IdentityCleanup(ctx, userID); err != nil {
			return nil, fmt.Errorf("identity cleanup: %w", err)
		}
	}
	if err := s.RevokeAllSessions(ctx, userID); err != nil {
		return nil, err
	}

	endsAt := time.Now().UTC().Add(gracePeriod)
	s.audit(ctx, &userID, nil, "account.deletion_requested", "", "", "", map[string]any{
		"grace_period_ends_at": endsAt,
	})

	return &AccountDeletionState{GracePeriodEndsAt: endsAt}, nil
}

// PurgeDeletedAccounts finds users whose deletion grace period has ended and
// transitions them to deleted. It is a minimal background-ready stub; callers
// are responsible for scheduling.
func (s *Service) PurgeDeletedAccounts(ctx context.Context) (int, error) {
	users, err := s.repo.ListUsersPastGracePeriod(ctx, time.Now().UTC())
	if err != nil {
		return 0, err
	}

	purged := 0
	for _, u := range users {
		if err := s.repo.MarkUserDeleted(ctx, u.ID); err != nil {
			return purged, fmt.Errorf("purge user %s: %w", u.ID, err)
		}
		s.audit(ctx, &u.ID, nil, "account.deleted", "", "", "", nil)
		purged++
	}
	return purged, nil
}

// ValidateAccessToken parses and validates a bearer access token.
func (s *Service) ValidateAccessToken(ctx context.Context, token string) (AccessTokenClaims, error) {
	claims := &accessClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.signer, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !parsed.Valid {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	if claims.Type != "access" {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	return AccessTokenClaims{UserID: claims.UserID, SessionID: claims.SessionID, IsStaff: claims.IsStaff}, nil
}

// issueAccessToken signs a short-lived JWT for the user/session pair.
func (s *Service) issueAccessToken(userID, sessionID string, isStaff bool) (string, int, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.cfg.AccessTokenTTL)
	claims := accessClaims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      "access",
		IsStaff:   isStaff,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.signer)
	if err != nil {
		return "", 0, err
	}
	return signed, int(s.cfg.AccessTokenTTL.Seconds()), nil
}

// newRefreshToken generates a random refresh token and returns it together
// with its SHA-256 hash for storage.
func (s *Service) newRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	hash = hashRefreshToken(token)
	return token, hash, nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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

func (s *Service) isStaff(email string) bool {
	if len(s.cfg.StaffEmails) == 0 {
		return false
	}
	normalized := NormalizeEmail(email)
	for _, e := range s.cfg.StaffEmails {
		if NormalizeEmail(e) == normalized {
			return true
		}
	}
	return false
}

func (s *Service) audit(ctx context.Context, userID, sessionID *string, eventType, ip, userAgent, fingerprint string, metadata map[string]any) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	_ = s.repo.CreateAuditEvent(ctx, &AuditEvent{
		UserID:      userID,
		SessionID:   sessionID,
		EventType:   eventType,
		IPAddress:   ip,
		UserAgent:   userAgent,
		Fingerprint: fingerprint,
		Metadata:    metadata,
	})
}
