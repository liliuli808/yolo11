package identity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/platform/config"
	"github.com/yiguan/api/internal/platform/httpx"
)

// Handler exposes identity endpoints over HTTP.
type Handler struct {
	service     *Service
	authHandler *auth.Handler
	cfg         *config.Config
}

// NewHandler creates an identity HTTP handler.
func NewHandler(service *Service, authHandler *auth.Handler, cfg *config.Config) *Handler {
	return &Handler{service: service, authHandler: authHandler, cfg: cfg}
}

// WithActivePersona returns a middleware stack that requires authentication and
// a usable default persona.
func (h *Handler) WithActivePersona() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return h.authHandler.AuthMiddleware(h.ActivePersonaMiddleware(next))
	}
}

// AuthMiddleware delegates to the auth handler's auth middleware.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return h.authHandler.AuthMiddleware(next)
}

// OptionalAuthMiddleware delegates to the auth handler's optional auth middleware.
func (h *Handler) OptionalAuthMiddleware(next http.Handler) http.Handler {
	return h.authHandler.OptionalAuthMiddleware(next)
}

// Mount registers identity routes on r.
func (h *Handler) Mount(r chi.Router) {
	// Authenticated /me endpoints.
	r.With(h.authHandler.AuthMiddleware).Get("/me", h.GetMe)
	r.With(h.authHandler.AuthMiddleware).Post("/me/email-change-requests", h.RequestEmailChange)
	r.With(h.authHandler.AuthMiddleware).Post("/me/email-change-confirmations", h.ConfirmEmailChange)
	r.With(h.authHandler.AuthMiddleware).Post("/me/export-requests", h.RequestDataExport)
	r.With(h.authHandler.AuthMiddleware).Get("/me/export-requests/{exportId}", h.GetDataExport)

	r.With(h.authHandler.AuthMiddleware).Get("/me/personas", h.ListMyPersonas)
	r.With(h.authHandler.AuthMiddleware).Post("/me/personas", h.CreatePersona)
	r.With(h.authHandler.AuthMiddleware).Get("/me/personas/{personaId}", h.GetMyPersona)
	r.With(h.authHandler.AuthMiddleware).Patch("/me/personas/{personaId}", h.UpdatePersona)
	r.With(h.authHandler.AuthMiddleware).Delete("/me/personas/{personaId}", h.ArchivePersona)
	r.With(h.authHandler.AuthMiddleware).Put("/me/personas/{personaId}/default", h.SetDefaultPersona)

	// Public persona endpoints.
	r.Get("/personas/{personaId}", h.GetPersona)
	r.Get("/personas/{personaId}/posts", h.ListPersonaPosts)
}

type realProfileResponse struct {
	ID               string                 `json:"id"`
	EmailMasked      string                 `json:"emailMasked"`
	Status           string                 `json:"status"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
	DefaultPersonaID *string                `json:"defaultPersonaId"`
	PersonaCount     int                    `json:"personaCount"`
	MaxPersonas      int                    `json:"maxPersonas"`
	Deletion         *deletionStateResponse `json:"deletion"`
}

type deletionStateResponse struct {
	GracePeriodEndsAt string `json:"gracePeriodEndsAt"`
}

type privatePersonaResponse struct {
	ID        string         `json:"id"`
	Alias     string         `json:"alias"`
	Bio       *string        `json:"bio"`
	Avatar    avatarResponse `json:"avatar"`
	CreatedAt string         `json:"createdAt"`
	UpdatedAt string         `json:"updatedAt"`
	NoteCount int            `json:"noteCount"`
	IsDefault bool           `json:"isDefault"`
	Status    string         `json:"status"`
}

type publicPersonaResponse struct {
	ID        string         `json:"id"`
	Alias     string         `json:"alias"`
	Bio       *string        `json:"bio"`
	Avatar    avatarResponse `json:"avatar"`
	CreatedAt string         `json:"createdAt"`
	NoteCount int            `json:"noteCount"`
	IsBlocked bool           `json:"isBlocked"`
}

type avatarResponse struct {
	Seed  string `json:"seed"`
	Color string `json:"color"`
}

type cursorPageResponse struct {
	Data       []any              `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

type paginationResponse struct {
	NextCursor *string `json:"nextCursor"`
	HasMore    bool    `json:"hasMore"`
	Limit      int     `json:"limit"`
}

type personaCreateRequest struct {
	Alias       string  `json:"alias"`
	Bio         *string `json:"bio"`
	AvatarSeed  string  `json:"avatarSeed"`
	AvatarColor string  `json:"avatarColor"`
}

type personaUpdateRequest struct {
	Alias       *string `json:"alias"`
	Bio         *string `json:"bio"`
	AvatarSeed  *string `json:"avatarSeed"`
	AvatarColor *string `json:"avatarColor"`
}

type emailChangeRequest struct {
	NewEmail string `json:"newEmail"`
}

type emailChangeConfirmation struct {
	NewEmail         string `json:"newEmail"`
	VerificationCode string `json:"verificationCode"`
}

type dataExportRequest struct {
	Format string `json:"format"`
}

type dataExportResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	Format      string  `json:"format"`
	RequestedAt string  `json:"requestedAt"`
	ReadyAt     *string `json:"readyAt"`
	DownloadURL *string `json:"downloadUrl"`
}

// GetMe handles GET /v1/me.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		httpError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
		return
	}

	profile, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toRealProfileResponse(profile)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// RequestEmailChange handles POST /v1/me/email-change-requests.
func (h *Handler) RequestEmailChange(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req emailChangeRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	if err := h.service.RequestEmailChange(r.Context(), userID, req.NewEmail, clientIP(r, h.cfg.RateLimitBehindProxy), r.Header.Get("X-Device-Fingerprint")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ConfirmEmailChange handles POST /v1/me/email-change-confirmations.
func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req emailChangeConfirmation
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	profile, err := h.service.ConfirmEmailChange(r.Context(), userID, req.NewEmail, req.VerificationCode)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toRealProfileResponse(profile)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// RequestDataExport handles POST /v1/me/export-requests.
func (h *Handler) RequestDataExport(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req dataExportRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	export, err := h.service.RequestDataExport(r.Context(), userID, req.Format)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusAccepted, h.toDataExportResponse(export)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetDataExport handles GET /v1/me/export-requests/{exportId}.
func (h *Handler) GetDataExport(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	exportID := chi.URLParam(r, "exportId")

	export, err := h.service.GetDataExport(r.Context(), userID, exportID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toDataExportResponse(export)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// ListMyPersonas handles GET /v1/me/personas.
func (h *Handler) ListMyPersonas(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}

	personas, err := h.service.ListPersonas(r.Context(), userID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	resp := make([]privatePersonaResponse, len(personas))
	for i, p := range personas {
		resp[i] = h.toPrivatePersonaResponse(p)
	}

	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// CreatePersona handles POST /v1/me/personas.
func (h *Handler) CreatePersona(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}

	var req personaCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	p, err := h.service.CreatePersona(r.Context(), userID, &PersonaCreateRequest{
		Alias:       req.Alias,
		Bio:         req.Bio,
		AvatarSeed:  req.AvatarSeed,
		AvatarColor: req.AvatarColor,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, h.toPrivatePersonaResponse(p)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetMyPersona handles GET /v1/me/personas/{personaId}.
func (h *Handler) GetMyPersona(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	personaID := chi.URLParam(r, "personaId")

	p, err := h.service.GetPrivatePersona(r.Context(), userID, personaID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toPrivatePersonaResponse(p)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// UpdatePersona handles PATCH /v1/me/personas/{personaId}.
func (h *Handler) UpdatePersona(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID := chi.URLParam(r, "personaId")

	var req personaUpdateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	p, err := h.service.UpdatePersona(r.Context(), userID, personaID, &PersonaUpdateRequest{
		Alias:       req.Alias,
		Bio:         req.Bio,
		AvatarSeed:  req.AvatarSeed,
		AvatarColor: req.AvatarColor,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toPrivatePersonaResponse(p)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// ArchivePersona handles DELETE /v1/me/personas/{personaId}.
func (h *Handler) ArchivePersona(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID := chi.URLParam(r, "personaId")

	if err := h.service.ArchivePersona(r.Context(), userID, personaID); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SetDefaultPersona handles PUT /v1/me/personas/{personaId}/default.
func (h *Handler) SetDefaultPersona(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID := chi.URLParam(r, "personaId")

	p, err := h.service.SetDefaultPersona(r.Context(), userID, personaID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toPrivatePersonaResponse(p)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetPersona handles GET /v1/personas/{personaId}.
func (h *Handler) GetPersona(w http.ResponseWriter, r *http.Request) {
	personaID := chi.URLParam(r, "personaId")

	p, err := h.service.GetPublicPersona(r.Context(), personaID)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	if err := httpx.WriteJSON(w, http.StatusOK, h.toPublicPersonaResponse(p)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// ListPersonaPosts handles GET /v1/personas/{personaId}/posts.
func (h *Handler) ListPersonaPosts(w http.ResponseWriter, r *http.Request) {
	personaID := chi.URLParam(r, "personaId")
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r)

	page, err := h.service.ListPersonaPosts(r.Context(), personaID, cursor, limit)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	resp := cursorPageResponse{
		Data: page.Data,
		Pagination: paginationResponse{
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
			Limit:      page.Limit,
		},
	}
	if resp.Data == nil {
		resp.Data = []any{}
	}

	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

func (h *Handler) requireUser(ctx context.Context, w http.ResponseWriter) (string, bool) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		httpError(ctx, w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
		return "", false
	}
	return userID, true
}

func (h *Handler) requireIdempotencyKey(r *http.Request, w http.ResponseWriter) (string, bool) {
	key := r.Header.Get("Idempotency-Key")
	if strings.TrimSpace(key) == "" || len(key) < 8 || len(key) > 128 {
		httpError(r.Context(), w, http.StatusBadRequest, "IDEMPOTENCY.MISSING_KEY", "Idempotency-Key header is required")
		return "", false
	}
	return key, true
}

func (h *Handler) respondDomainError(ctx context.Context, w http.ResponseWriter, err error) {
	var rateLimitErr *auth.RateLimitError
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpError(ctx, w, http.StatusNotFound, "ME.NOT_FOUND", "account not found")
	case errors.Is(err, ErrInvalidEmail):
		httpError(ctx, w, http.StatusBadRequest, "VALIDATION_FAILED", "please enter a valid email address")
	case errors.Is(err, ErrEmailAlreadyUsed):
		httpError(ctx, w, http.StatusConflict, "ME.EMAIL_ALREADY_USED", "that email address is already in use")
	case errors.Is(err, ErrInvalidCode):
		httpError(ctx, w, http.StatusForbidden, "ME.EMAIL_CHANGE_INVALID_CODE", "the verification code is incorrect or expired")
	case errors.As(err, &rateLimitErr):
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rateLimitErr.RetryAfter.Seconds())))
		httpError(ctx, w, http.StatusTooManyRequests, "ME.EMAIL_CHANGE_RATE_LIMITED", "too many email change requests. please try again later")
	case errors.Is(err, ErrInvalidExportFormat):
		httpError(ctx, w, http.StatusUnprocessableEntity, "VALIDATION.EXPORT_FORMAT_INVALID", "unsupported export format")
	case errors.Is(err, ErrExportRateLimited):
		httpError(ctx, w, http.StatusTooManyRequests, "ME.EXPORT_RATE_LIMITED", "an export was requested recently; please try again later")
	case errors.Is(err, ErrExportNotFound):
		httpError(ctx, w, http.StatusNotFound, "ME.EXPORT_NOT_FOUND", "export request not found")
	case errors.Is(err, ErrPersonaNotFound):
		httpError(ctx, w, http.StatusNotFound, "PERSONA.NOT_FOUND", "persona not found")
	case errors.Is(err, ErrPersonaMaxReached):
		httpError(ctx, w, http.StatusForbidden, "PERSONA.MAX_REACHED", "maximum number of personas reached")
	case errors.Is(err, ErrPersonaAliasTaken):
		httpError(ctx, w, http.StatusConflict, "PERSONA.ALIAS_TAKEN", "that alias is already in use")
	case errors.Is(err, ErrPersonaAliasDisallowed):
		httpError(ctx, w, http.StatusUnprocessableEntity, "PERSONA.ALIAS_DISALLOWED", "invalid persona fields")
	case errors.Is(err, ErrPersonaArchived):
		httpError(ctx, w, http.StatusForbidden, "PERSONA.ARCHIVED", "persona is archived")
	case errors.Is(err, ErrPersonaRestricted):
		httpError(ctx, w, http.StatusForbidden, "PERSONA.RESTRICTED", "persona cannot be used")
	default:
		httpError(ctx, w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
	}
}

func (h *Handler) toRealProfileResponse(p *RealProfile) realProfileResponse {
	resp := realProfileResponse{
		ID:               p.ID,
		EmailMasked:      MaskEmail(p.EmailNormalized),
		Status:           p.Status,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
		DefaultPersonaID: p.DefaultPersonaID,
		PersonaCount:     p.PersonaCount,
		MaxPersonas:      p.MaxPersonas,
	}
	if p.DeletionGracePeriodEndsAt != nil {
		resp.Deletion = &deletionStateResponse{
			GracePeriodEndsAt: p.DeletionGracePeriodEndsAt.Format(time.RFC3339),
		}
	}
	return resp
}

func (h *Handler) toPrivatePersonaResponse(p *PrivatePersona) privatePersonaResponse {
	return privatePersonaResponse{
		ID:        p.ID,
		Alias:     p.Alias,
		Bio:       p.Bio,
		Avatar:    avatarResponse{Seed: p.AvatarSeed, Color: p.AvatarColor},
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
		NoteCount: p.NoteCount,
		IsDefault: p.IsDefault,
		Status:    p.Status,
	}
}

func (h *Handler) toPublicPersonaResponse(p *PublicPersona) publicPersonaResponse {
	return publicPersonaResponse{
		ID:        p.ID,
		Alias:     p.Alias,
		Bio:       p.Bio,
		Avatar:    avatarResponse{Seed: p.AvatarSeed, Color: p.AvatarColor},
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		NoteCount: p.NoteCount,
		IsBlocked: p.IsBlocked,
	}
}

func (h *Handler) toDataExportResponse(e *DataExport) dataExportResponse {
	resp := dataExportResponse{
		ID:          e.ID,
		Status:      e.Status,
		Format:      e.Format,
		RequestedAt: e.RequestedAt.Format(time.RFC3339),
	}
	if e.ReadyAt != nil {
		t := e.ReadyAt.Format(time.RFC3339)
		resp.ReadyAt = &t
	}
	if e.DownloadURL != nil {
		resp.DownloadURL = e.DownloadURL
	}
	return resp
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	httpx.Error(ctx, w, status, code, message)
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

func parseLimit(r *http.Request) int {
	q := r.URL.Query().Get("limit")
	if q == "" {
		return 20
	}
	n, err := strconv.Atoi(q)
	if err != nil || n < 1 {
		return 20
	}
	if n > 100 {
		return 100
	}
	return n
}
