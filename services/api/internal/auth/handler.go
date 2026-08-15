package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/yiguan/api/internal/platform/config"
	"github.com/yiguan/api/internal/platform/httpx"
)

const refreshTokenCookie = "refresh_token"

type contextKey int

const (
	userIDKey contextKey = iota
	sessionIDKey
	isStaffKey
)

// Handler exposes auth endpoints over HTTP.
type Handler struct {
	service *Service
	cfg     *config.Config
}

// NewHandler creates an auth HTTP handler.
func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

// Mount registers auth routes on r.
func (h *Handler) Mount(r chi.Router) {
	r.Post("/auth/email-codes", h.SendEmailCode)
	r.Post("/auth/email-sessions", h.CreateEmailSession)
	r.Post("/auth/refresh", h.RefreshSession)

	r.With(h.AuthMiddleware).Delete("/auth/session", h.DeleteSession)
	r.With(h.AuthMiddleware).Delete("/me", h.DeleteMe)
}

type emailCodeRequest struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

type emailSessionRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type accountDeletionRequest struct {
	VerificationCode string `json:"verificationCode"`
}

type sessionResponse struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	TokenType    string  `json:"tokenType"`
	ExpiresIn    int     `json:"expiresIn"`
	PersonaID    *string `json:"personaId"`
	IsStaff      bool    `json:"isStaff"`
}

type accessTokenResponse struct {
	AccessToken string  `json:"accessToken"`
	TokenType   string  `json:"tokenType"`
	ExpiresIn   int     `json:"expiresIn"`
	PersonaID   *string `json:"personaId"`
	IsStaff     bool    `json:"isStaff"`
}

type accountDeletionResponse struct {
	GracePeriodEndsAt string `json:"gracePeriodEndsAt"`
}

// SendEmailCode handles POST /v1/auth/email-codes.
func (h *Handler) SendEmailCode(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req emailCodeRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	purpose := strings.TrimSpace(req.Purpose)
	if purpose == "" {
		purpose = "login"
	}
	if purpose != "login" && purpose != "email_change" && purpose != "deletion" {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid purpose")
		return
	}

	ip := clientIP(r, h.cfg.RateLimitBehindProxy)
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	if err := h.service.RequestEmailCode(r.Context(), req.Email, purpose, ip, fingerprint); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateEmailSession handles POST /v1/auth/email-sessions.
func (h *Handler) CreateEmailSession(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req emailSessionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	ip := clientIP(r, h.cfg.RateLimitBehindProxy)
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	tokens, err := h.service.CreateEmailSession(r.Context(), req.Email, req.Code, ip, fingerprint, r.UserAgent())
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	h.setRefreshTokenCookie(w, tokens.RefreshToken, int(h.cfg.RefreshTokenTTL.Seconds()))

	if err := httpx.WriteJSON(w, http.StatusCreated, sessionResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		PersonaID:    nil,
		IsStaff:      tokens.IsStaff,
	}); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// RefreshSession handles POST /v1/auth/refresh.
func (h *Handler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req refreshTokenRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if c, err := r.Cookie(refreshTokenCookie); err == nil && c.Value != "" {
			refreshToken = c.Value
		}
	}

	ip := clientIP(r, h.cfg.RateLimitBehindProxy)
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	result, err := h.service.RefreshSession(r.Context(), refreshToken, ip, fingerprint, r.UserAgent())
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken, int(h.cfg.RefreshTokenTTL.Seconds()))

	if err := httpx.WriteJSON(w, http.StatusOK, accessTokenResponse{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresIn:   result.ExpiresIn,
		PersonaID:   nil,
		IsStaff:     result.IsStaff,
	}); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// DeleteSession handles DELETE /v1/auth/session.
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	sessionID := SessionIDFromContext(r.Context())
	if sessionID == "" {
		httpError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing session")
		return
	}

	if err := h.service.Logout(r.Context(), sessionID); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	h.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// DeleteMe handles DELETE /v1/me.
func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req accountDeletionRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		httpError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
		return
	}

	user, err := h.service.repo.GetUserByID(r.Context(), userID)
	if err != nil {
		h.respondDomainError(r.Context(), w, fmt.Errorf("%w: %v", ErrUserNotFound, err))
		return
	}

	ip := clientIP(r, h.cfg.RateLimitBehindProxy)
	fingerprint := r.Header.Get("X-Device-Fingerprint")

	if err := h.service.VerifyEmailCode(r.Context(), user.EmailNormalized, req.VerificationCode, "deletion"); err != nil {
		h.respondDomainError(r.Context(), w, ErrDeletionInvalidCode)
		return
	}

	state, err := h.service.RequestAccountDeletion(r.Context(), userID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	h.clearRefreshTokenCookie(w)
	h.service.audit(r.Context(), &userID, nil, "account.deletion_confirmed", ip, r.UserAgent(), fingerprint, nil)

	if err := httpx.WriteJSON(w, http.StatusOK, accountDeletionResponse{
		GracePeriodEndsAt: state.GracePeriodEndsAt.Format("2006-01-02T15:04:05Z07:00"),
	}); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// AuthMiddleware validates the bearer access token and injects the user and
// session identifiers into the request context.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			httpError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			httpError(r.Context(), w, http.StatusUnauthorized, "AUTH.INVALID_TOKEN", "invalid authorization header")
			return
		}

		claims, err := h.service.ValidateAccessToken(r.Context(), parts[1])
		if err != nil {
			h.respondDomainError(r.Context(), w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, sessionIDKey, claims.SessionID)
		ctx = context.WithValue(ctx, isStaffKey, claims.IsStaff)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware validates a bearer access token if one is present and
// continues the request regardless. If the header is missing, the request is
// treated as unauthenticated. If the header is malformed or the token is
// invalid/expired, it returns 401.
func (h *Handler) OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			httpError(r.Context(), w, http.StatusUnauthorized, "AUTH.INVALID_TOKEN", "invalid authorization header")
			return
		}

		claims, err := h.service.ValidateAccessToken(r.Context(), parts[1])
		if err != nil {
			h.respondDomainError(r.Context(), w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, sessionIDKey, claims.SessionID)
		ctx = context.WithValue(ctx, isStaffKey, claims.IsStaff)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// IsStaffFromContext returns true if the authenticated principal is a staff member.
func IsStaffFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(isStaffKey).(bool)
	return v
}

// StaffAuthMiddleware requires a valid bearer token with the staff claim set to true.
func (h *Handler) StaffAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			httpError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			httpError(r.Context(), w, http.StatusUnauthorized, "AUTH.INVALID_TOKEN", "invalid authorization header")
			return
		}

		claims, err := h.service.ValidateAccessToken(r.Context(), parts[1])
		if err != nil {
			h.respondDomainError(r.Context(), w, err)
			return
		}
		if !claims.IsStaff {
			httpError(r.Context(), w, http.StatusForbidden, "MODERATION.NOT_MODERATOR", "staff access required")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, sessionIDKey, claims.SessionID)
		ctx = context.WithValue(ctx, isStaffKey, claims.IsStaff)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SessionIDFromContext returns the authenticated session ID, if any.
func SessionIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionIDKey).(string)
	return v
}

func (h *Handler) respondDomainError(ctx context.Context, w http.ResponseWriter, err error) {
	var rateLimitErr *RateLimitError
	switch {
	case errors.Is(err, ErrInvalidEmail):
		httpError(ctx, w, http.StatusBadRequest, "AUTH.INVALID_EMAIL", "please enter a valid email address")
	case errors.Is(err, ErrInvalidCode):
		httpError(ctx, w, http.StatusUnauthorized, "AUTH.INVALID_CODE", "the code is incorrect or expired")
	case errors.As(err, &rateLimitErr):
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rateLimitErr.RetryAfter.Seconds())))
		httpError(ctx, w, http.StatusTooManyRequests, "AUTH.RATE_LIMITED", "too many attempts. please try again later")
	case errors.Is(err, ErrSessionExpired):
		httpError(ctx, w, http.StatusUnauthorized, "AUTH.SESSION_EXPIRED", "your session expired. please sign in again")
	case errors.Is(err, ErrSessionRevoked):
		httpError(ctx, w, http.StatusUnauthorized, "AUTH.SESSION_REVOKED", "this session was signed out")
	case errors.Is(err, ErrInvalidToken):
		httpError(ctx, w, http.StatusUnauthorized, "AUTH.INVALID_TOKEN", "please sign in again")
	case errors.Is(err, ErrUserNotFound):
		httpError(ctx, w, http.StatusNotFound, "ME.NOT_FOUND", "account not found")
	case errors.Is(err, ErrDeletionAlreadyPending):
		httpError(ctx, w, http.StatusConflict, "ME.DELETION_ALREADY_PENDING", "account deletion is already in progress")
	case errors.Is(err, ErrDeletionInvalidCode):
		httpError(ctx, w, http.StatusForbidden, "ME.DELETION_INVALID_CODE", "the confirmation code is incorrect or expired")
	default:
		httpError(ctx, w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
	}
}

func (h *Handler) requireIdempotencyKey(r *http.Request, w http.ResponseWriter) (string, bool) {
	key := r.Header.Get("Idempotency-Key")
	if strings.TrimSpace(key) == "" || len(key) < 8 || len(key) > 128 {
		httpError(r.Context(), w, http.StatusBadRequest, "IDEMPOTENCY.MISSING_KEY", "Idempotency-Key header is required")
		return "", false
	}
	return key, true
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	httpx.Error(ctx, w, status, code, message)
}

func (h *Handler) setRefreshTokenCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.cfg.Environment == "production",
	})
}

func (h *Handler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.cfg.Environment == "production",
	})
}

func clientIP(r *http.Request, behindProxy bool) string {
	if behindProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			var ip string
			if i := strings.LastIndex(xff, ","); i >= 0 {
				ip = strings.TrimSpace(xff[i+1:])
			} else {
				ip = strings.TrimSpace(xff)
			}
			if ip != "" {
				return stripHostPort(ip)
			}
		}
		if xri := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xri != "" {
			return stripHostPort(xri)
		}
	}
	return stripHostPort(r.RemoteAddr)
}

func stripHostPort(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
