package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yiguan/api/internal/platform/config"
	"golang.org/x/crypto/bcrypt"
)

// Common auth domain errors.
var (
	ErrInvalidEmail            = errors.New("invalid email")
	ErrInvalidCode             = errors.New("invalid or expired verification code")
	ErrRateLimited             = errors.New("rate limited")
	ErrSessionExpired          = errors.New("session expired")
	ErrSessionRevoked          = errors.New("session revoked")
	ErrInvalidToken            = errors.New("invalid token")
	ErrUserNotFound            = errors.New("user not found")
	ErrDeletionAlreadyPending  = errors.New("account deletion already pending")
	ErrDeletionInvalidCode     = errors.New("invalid deletion verification code")
	ErrDeletionInvalidPassword = errors.New("invalid deletion password")
	ErrUsernameTaken           = errors.New("username already taken")
	ErrInvalidCredentials      = errors.New("invalid username or password")
	ErrCaptchaFailed           = errors.New("captcha verification failed")
	ErrInvalidUsername         = errors.New("invalid username")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrInviteCodeInvalid       = errors.New("invalid invite code")
	ErrInviteCodeUsed          = errors.New("invite code already used")
	ErrInviteCodeExpired       = errors.New("invite code expired")
	ErrInviteCodeNotFound      = errors.New("invite code not found")
)

// dummyPasswordHash is a precomputed bcrypt hash used in the nil-user Login
// branch to burn roughly the same time as a real password comparison, avoiding
// latency-based username enumeration.
var dummyPasswordHash = func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("lantern-dummy-password"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return h
}()

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
	// SessionID identifies the created session.
	SessionID string
	// UserID identifies the authenticated user.
	UserID string
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
	cfg       *config.Config
	repo      Repository
	mailer    Mailer
	limiter   RateLimiter
	turnstile TurnstileVerifier
	signer    []byte
	codeKey   []byte
	keys      keyBuilder

	// IdentityCleanup is an optional hook invoked when an account enters the
	// deletion grace period. It is used by the identity subsystem to archive
	// personas and clear default persona references.
	IdentityCleanup func(ctx context.Context, userID string) error
}

// NewService creates a new auth Service.
func NewService(cfg *config.Config, repo Repository, mailer Mailer, limiter RateLimiter, turnstile TurnstileVerifier) *Service {
	return &Service{
		cfg:       cfg,
		repo:      repo,
		mailer:    mailer,
		limiter:   limiter,
		turnstile: turnstile,
		signer:    []byte(cfg.JWTSigningKey),
		codeKey:   []byte(cfg.EmailCodeKey),
	}
}

// Predefined rate limits. These apply independently by scope.
var (
	registerByUsername = RateLimit{Count: 5, Window: 10 * time.Minute}
	registerByIP       = RateLimit{Count: 20, Window: 10 * time.Minute}
	registerByFP       = RateLimit{Count: 10, Window: 10 * time.Minute}

	loginByUsername = RateLimit{Count: 10, Window: 10 * time.Minute}
	loginByIP       = RateLimit{Count: 40, Window: 10 * time.Minute}
	loginByFP       = RateLimit{Count: 20, Window: 10 * time.Minute}
)

// Register creates a user with username/password and a session.
func (s *Service) Register(ctx context.Context, username, password, turnstileToken, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	if err := s.verifyTurnstile(ctx, turnstileToken); err != nil {
		return nil, err
	}
	normalized := NormalizeUsername(username)
	if err := validateUsername(normalized); err != nil {
		return nil, ErrInvalidUsername
	}
	if err := validatePassword(password); err != nil {
		return nil, ErrInvalidPassword
	}

	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.username("register", normalized), registerByUsername); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.ip("register", ip), registerByIP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.fingerprint("register", fingerprint), registerByFP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	existing, err := s.repo.GetUserByUsername(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if existing != nil {
		return nil, ErrUsernameTaken
	}

	sum := sha512.Sum384([]byte(password))
	hash, err := bcrypt.GenerateFromPassword(sum[:], bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, normalized, string(hash))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	tokens, err := s.createSession(ctx, user.ID, s.isStaff(user.Username), ip, fingerprint, userAgent)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &user.ID, &tokens.SessionID, "account.registered", ip, userAgent, fingerprint, nil)
	return tokens, nil
}

// Login validates credentials and creates a session.
func (s *Service) Login(ctx context.Context, username, password, turnstileToken, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	if err := s.verifyTurnstile(ctx, turnstileToken); err != nil {
		return nil, err
	}
	normalized := NormalizeUsername(username)
	if err := validateUsername(normalized); err != nil {
		return nil, ErrInvalidCredentials
	}
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.username("login", normalized), loginByUsername); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.ip("login", ip), loginByIP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}
	if allowed, retryAfter, err := s.limiter.Allow(ctx, s.keys.fingerprint("login", fingerprint), loginByFP); err != nil || !allowed {
		return nil, &RateLimitError{RetryAfter: retryAfter}
	}

	user, err := s.repo.GetUserByUsername(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		sum := sha512.Sum384([]byte(password))
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, sum[:])
		return nil, ErrInvalidCredentials
	}
	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
	sum := sha512.Sum384([]byte(password))
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), sum[:]) != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.createSession(ctx, user.ID, s.isStaff(user.Username), ip, fingerprint, userAgent)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &user.ID, &tokens.SessionID, "session.created", ip, userAgent, fingerprint, map[string]any{"is_staff": s.isStaff(user.Username)})
	return tokens, nil
}

// createSession builds a session for the user and returns access + refresh tokens.
func (s *Service) createSession(ctx context.Context, userID string, isStaff bool, ip, fingerprint, userAgent string) (*TokenResponse, error) {
	refreshToken, refreshHash, err := s.newRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	session := &Session{
		UserID:           userID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        now.Add(s.cfg.RefreshTokenTTL),
		IPAddress:        ip,
		UserAgent:        userAgent,
		Fingerprint:      fingerprint,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	accessToken, expiresIn, err := s.issueAccessToken(userID, session.ID, isStaff)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		IsStaff:      isStaff,
		SessionID:    session.ID,
		UserID:       userID,
	}, nil
}

// VerifyPasswordForDeletion checks the account password to confirm deletion.
func (s *Service) VerifyPasswordForDeletion(ctx context.Context, userID, password string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUserNotFound, err)
	}
	if user.PasswordHash == "" {
		return ErrDeletionInvalidPassword
	}
	sum := sha512.Sum384([]byte(password))
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), sum[:]) != nil {
		return ErrDeletionInvalidPassword
	}
	return nil
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

	accessToken, expiresIn, err := s.issueAccessToken(session.UserID, newSession.ID, s.isStaff(user.Username))
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	s.audit(ctx, &session.UserID, &newSession.ID, "session.rotated", ip, userAgent, fingerprint, nil)

	return &RefreshResult{
		AccessTokenResponse: AccessTokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   expiresIn,
			IsStaff:     s.isStaff(user.Username),
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

func hashEmailCode(key []byte, code string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func hmacEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

var usernameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,19}$`)

// NormalizeUsername lowercases and trims a username.
func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func validateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	return nil
}

func (s *Service) isStaff(username string) bool {
	if len(s.cfg.StaffUsernames) == 0 {
		return false
	}
	normalized := NormalizeUsername(username)
	for _, u := range s.cfg.StaffUsernames {
		if NormalizeUsername(u) == normalized {
			return true
		}
	}
	return false
}

func (s *Service) verifyTurnstile(ctx context.Context, token string) error {
	if s.turnstile == nil {
		return nil
	}
	return s.turnstile.Verify(ctx, token)
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
