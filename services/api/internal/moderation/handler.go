package moderation

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/identity"
	"github.com/yiguan/api/internal/platform/config"
	"github.com/yiguan/api/internal/platform/httpx"
)

// Handler exposes moderation endpoints over HTTP.
type Handler struct {
	service     *Service
	authHandler *auth.Handler
	idHandler   *identity.Handler
	idService   *identity.Service
	cfg         *config.Config
}

// NewHandler creates a moderation HTTP handler.
func NewHandler(service *Service, authHandler *auth.Handler, idHandler *identity.Handler, idService *identity.Service, cfg *config.Config) *Handler {
	return &Handler{service: service, authHandler: authHandler, idHandler: idHandler, idService: idService, cfg: cfg}
}

// Mount registers moderation routes on r.
func (h *Handler) Mount(r chi.Router) {
	// User-facing blocks and reports.
	r.With(h.idMiddleware()).Get("/me/blocks", h.ListBlocks)
	r.With(h.idMiddleware()).Post("/me/blocks", h.CreateBlock)
	r.With(h.idMiddleware()).Delete("/me/blocks/{blockId}", h.DeleteBlock)

	r.With(h.authHandler.AuthMiddleware).Post("/reports", h.CreateReport)
	r.With(h.authHandler.AuthMiddleware).Get("/reports/{reportId}", h.GetReport)

	// Staff-only moderation console endpoints.
	r.With(h.authHandler.StaffAuthMiddleware).Get("/moderation/cases", h.ListCases)
	r.With(h.authHandler.StaffAuthMiddleware).Post("/moderation/cases", h.CreateCase)
	r.With(h.authHandler.StaffAuthMiddleware).Get("/moderation/cases/{caseId}", h.GetCase)
	r.With(h.authHandler.StaffAuthMiddleware).Patch("/moderation/cases/{caseId}", h.UpdateCase)
	r.With(h.authHandler.StaffAuthMiddleware).Get("/moderation/cases/{caseId}/actions", h.ListCaseActions)
}

func (h *Handler) idMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return h.idHandler.AuthMiddleware(h.idHandler.ActivePersonaMiddleware(next))
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

func (h *Handler) requireActivePersona(ctx context.Context, w http.ResponseWriter) (string, bool) {
	personaID := identity.ActivePersonaIDFromContext(ctx)
	if personaID == "" {
		httpError(ctx, w, http.StatusBadRequest, "PERSONA.DEFAULT_REQUIRED", "please select a default persona first")
		return "", false
	}
	return personaID, true
}

func (h *Handler) requireIdempotencyKey(r *http.Request, w http.ResponseWriter) (string, bool) {
	key := r.Header.Get("Idempotency-Key")
	if strings.TrimSpace(key) == "" || len(key) < 8 || len(key) > 128 {
		httpError(r.Context(), w, http.StatusBadRequest, "IDEMPOTENCY.MISSING_KEY", "Idempotency-Key header is required")
		return "", false
	}
	return key, true
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

type blockResponse struct {
	ID        string                `json:"id"`
	Persona   personaResponse       `json:"persona"`
	CreatedAt string                `json:"createdAt"`
}

type personaResponse struct {
	ID        string         `json:"id"`
	Alias     string         `json:"alias"`
	Bio       *string        `json:"bio"`
	Avatar    avatarResponse `json:"avatar"`
	CreatedAt string         `json:"createdAt"`
	NoteCount int            `json:"noteCount"`
}

type avatarResponse struct {
	Seed  string `json:"seed"`
	Color string `json:"color"`
}

type blockCreateRequest struct {
	PersonaID string `json:"personaId"`
}

type reportResponse struct {
	ID         string  `json:"id"`
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	Category   string  `json:"category"`
	Details    *string `json:"details"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"createdAt"`
	ResolvedAt *string `json:"resolvedAt"`
}

type reportCreateRequest struct {
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	Category   string  `json:"category"`
	Details    *string `json:"details"`
}

type moderationCaseResponse struct {
	ID         string  `json:"id"`
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	ReportIDs  []string `json:"reportIds"`
	Status     string  `json:"status"`
	Outcome    *string `json:"outcome"`
	Notes      *string `json:"notes"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

type moderationCaseCreateRequest struct {
	TargetType string   `json:"targetType"`
	TargetID   string   `json:"targetId"`
	ReportIDs  []string `json:"reportIds"`
}

type moderationCaseUpdateRequest struct {
	Status  string  `json:"status"`
	Outcome *string `json:"outcome"`
	Notes   *string `json:"notes"`
}

type moderationActionResponse struct {
	ID             string  `json:"id"`
	ModeratorUserID *string `json:"moderatorUserId"`
	ActionType     string  `json:"actionType"`
	TargetType     *string `json:"targetType"`
	TargetID       *string `json:"targetId"`
	Note           *string `json:"note"`
	CreatedAt      string  `json:"createdAt"`
}

// ListBlocks handles GET /v1/me/blocks.
func (h *Handler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	page, err := h.service.ListBlocks(r.Context(), personaID, r.URL.Query().Get("cursor"), parseLimit(r))
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	resp := cursorPageResponse{Data: []any{}, Pagination: paginationResponse{NextCursor: page.NextCursor, HasMore: page.HasMore, Limit: page.Limit}}
	for _, b := range page.Data {
		resp.Data = append(resp.Data, h.toBlockResponse(&b))
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// CreateBlock handles POST /v1/me/blocks.
func (h *Handler) CreateBlock(w http.ResponseWriter, r *http.Request) {
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	var req blockCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	b, err := h.service.CreateBlock(r.Context(), personaID, &CreateBlockRequest{BlockedPersonaID: req.PersonaID})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusCreated, h.toBlockResponse(b)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// DeleteBlock handles DELETE /v1/me/blocks/{blockId}.
func (h *Handler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	if err := h.service.DeleteBlock(r.Context(), chi.URLParam(r, "blockId"), personaID); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateReport handles POST /v1/reports.
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	var req reportCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	report, err := h.service.CreateReport(r.Context(), userID, &CreateReportRequest{
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Category:   req.Category,
		Details:    req.Details,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusCreated, h.toReportResponse(report)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetReport handles GET /v1/reports/{reportId}.
func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	report, err := h.service.GetReport(r.Context(), userID, chi.URLParam(r, "reportId"))
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toReportResponse(report)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// ListCases handles GET /v1/moderation/cases.
func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page, err := h.service.ListCases(r.Context(), status, r.URL.Query().Get("cursor"), parseLimit(r))
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	resp := cursorPageResponse{Data: []any{}, Pagination: paginationResponse{NextCursor: page.NextCursor, HasMore: page.HasMore, Limit: page.Limit}}
	for _, c := range page.Data {
		resp.Data = append(resp.Data, h.toCaseResponse(&c))
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// CreateCase handles POST /v1/moderation/cases.
func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	moderatorID := auth.UserIDFromContext(r.Context())
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	var req moderationCaseCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	c, err := h.service.CreateCase(r.Context(), moderatorID, &CreateCaseRequest{
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		ReportIDs:  req.ReportIDs,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusCreated, h.toCaseResponse(c)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetCase handles GET /v1/moderation/cases/{caseId}.
func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.GetCase(r.Context(), chi.URLParam(r, "caseId"))
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toCaseResponse(c)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// UpdateCase handles PATCH /v1/moderation/cases/{caseId}.
func (h *Handler) UpdateCase(w http.ResponseWriter, r *http.Request) {
	moderatorID := auth.UserIDFromContext(r.Context())
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	var req moderationCaseUpdateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	c, err := h.service.UpdateCase(r.Context(), moderatorID, chi.URLParam(r, "caseId"), &UpdateCaseRequest{
		Status:  req.Status,
		Outcome: req.Outcome,
		Notes:   req.Notes,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toCaseResponse(c)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// ListCaseActions handles GET /v1/moderation/cases/{caseId}/actions.
func (h *Handler) ListCaseActions(w http.ResponseWriter, r *http.Request) {
	actions, err := h.service.ListCaseActions(r.Context(), chi.URLParam(r, "caseId"))
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	resp := make([]moderationActionResponse, len(actions))
	for i, a := range actions {
		resp[i] = h.toActionResponse(&a)
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

func (h *Handler) toBlockResponse(b *BlockPublic) blockResponse {
	return blockResponse{
		ID:      b.ID,
		Persona: h.toPersonaResponse(b.Persona),
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) toPersonaResponse(p *identity.Persona) personaResponse {
	return personaResponse{
		ID:        p.ID,
		Alias:     p.Alias,
		Bio:       p.Bio,
		Avatar:    avatarResponse{Seed: p.AvatarSeed, Color: p.AvatarColor},
		CreatedAt: p.CreatedAt.Format(time.RFC3339),
		NoteCount: p.NoteCount,
	}
}

func (h *Handler) toReportResponse(r *ReportPublic) reportResponse {
	resp := reportResponse{
		ID:         r.ID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Category:   r.Category,
		Details:    r.Details,
		Status:     r.Status,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
	if r.ResolvedAt != nil {
		t := r.ResolvedAt.Format(time.RFC3339)
		resp.ResolvedAt = &t
	}
	return resp
}

func (h *Handler) toCaseResponse(c *ModerationCasePublic) moderationCaseResponse {
	return moderationCaseResponse{
		ID:         c.ID,
		TargetType: c.TargetType,
		TargetID:   c.TargetID,
		ReportIDs:  c.ReportIDs,
		Status:     c.Status,
		Outcome:    c.Outcome,
		Notes:      c.Notes,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) toActionResponse(a *ModerationActionPublic) moderationActionResponse {
	return moderationActionResponse{
		ID:              a.ID,
		ModeratorUserID: a.ModeratorUserID,
		ActionType:      a.ActionType,
		TargetType:      a.TargetType,
		TargetID:        a.TargetID,
		Note:            a.Note,
		CreatedAt:       a.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) respondDomainError(ctx context.Context, w http.ResponseWriter, err error) {
	var rateLimitErr *auth.RateLimitError
	switch {
	case errors.Is(err, ErrBlockNotFound):
		httpError(ctx, w, http.StatusNotFound, "BLOCK.NOT_FOUND", "block not found")
	case errors.Is(err, ErrBlockAlreadyExists):
		httpError(ctx, w, http.StatusConflict, "BLOCK.ALREADY_EXISTS", "you've already blocked this persona")
	case errors.Is(err, ErrBlockSelf):
		httpError(ctx, w, http.StatusUnprocessableEntity, "BLOCK.SELF", "you can't block yourself")
	case errors.Is(err, ErrReportNotFound):
		httpError(ctx, w, http.StatusNotFound, "REPORT.NOT_FOUND", "report not found")
	case errors.Is(err, ErrReportDuplicate):
		httpError(ctx, w, http.StatusConflict, "REPORT.DUPLICATE", "you've already reported this content")
	case errors.Is(err, ErrReportInvalidTarget):
		httpError(ctx, w, http.StatusUnprocessableEntity, "REPORT.INVALID_TARGET", "that report target isn't valid")
	case errors.Is(err, ErrReportTargetNotFound):
		httpError(ctx, w, http.StatusNotFound, "REPORT.TARGET_NOT_FOUND", "the reported content isn't available")
	case errors.Is(err, ErrReportSelf):
		httpError(ctx, w, http.StatusUnprocessableEntity, "REPORT.SELF", "you can't report your own content")
	case errors.Is(err, ErrCaseNotFound):
		httpError(ctx, w, http.StatusNotFound, "MODERATION.CASE_NOT_FOUND", "case not found")
	case errors.Is(err, ErrCaseInvalidOutcome):
		httpError(ctx, w, http.StatusUnprocessableEntity, "MODERATION.INVALID_OUTCOME", "that moderation outcome isn't valid")
	case errors.Is(err, ErrCaseInvalidStatus):
		httpError(ctx, w, http.StatusUnprocessableEntity, "MODERATION.INVALID_STATUS", "that case status isn't valid")
	case errors.As(err, &rateLimitErr):
		w.Header().Set("Retry-After", strconv.Itoa(int(rateLimitErr.RetryAfter.Seconds())))
		httpError(ctx, w, http.StatusTooManyRequests, "MODERATION.RATE_LIMITED", "you're submitting too quickly. please slow down")
	default:
		httpError(ctx, w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
	}
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	httpx.Error(ctx, w, status, code, message)
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
