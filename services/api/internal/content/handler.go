package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// Handler exposes content endpoints over HTTP.
type Handler struct {
	service   *Service
	idHandler *identity.Handler
	idService *identity.Service
	cfg       *config.Config
}

// NewHandler creates a content HTTP handler.
func NewHandler(service *Service, idHandler *identity.Handler, idService *identity.Service, cfg *config.Config) *Handler {
	return &Handler{service: service, idHandler: idHandler, idService: idService, cfg: cfg}
}

// Mount registers content routes on r.
func (h *Handler) Mount(r chi.Router) {
	opt := h.optionalAuthMiddleware()

	// Public read endpoints (optional authentication).
	r.With(opt).Get("/topics", h.ListTopics)
	r.With(opt).Get("/topics/{topicId}", h.GetTopic)
	r.With(opt).Get("/topics/{topicId}/posts", h.ListTopicPosts)

	r.With(opt).Get("/posts", h.ListPosts)
	r.With(opt).Get("/posts/{postId}", h.GetPost)
	r.With(opt).Get("/posts/{postId}/comments", h.ListComments)
	r.With(opt).Get("/posts/{postId}/comments/{commentId}", h.GetComment)
	r.Get("/posts/{postId}/reactions", h.ListReactions)

	r.With(opt).Get("/personas/{personaId}/posts", h.ListPersonaPosts)

	// Authenticated mutating endpoints.
	r.With(h.idMiddleware()).Post("/topics/{topicId}/follow", h.FollowTopic)
	r.With(h.idMiddleware()).Delete("/topics/{topicId}/follow", h.UnfollowTopic)

	r.With(h.idMiddleware()).Post("/posts", h.CreatePost)
	r.With(h.idMiddleware()).Patch("/posts/{postId}", h.UpdatePost)
	r.With(h.idMiddleware()).Delete("/posts/{postId}", h.DeletePost)

	r.With(h.idMiddleware()).Post("/posts/{postId}/saves", h.CreateSave)
	r.With(h.idMiddleware()).Delete("/posts/{postId}/saves", h.DeleteSave)

	r.With(h.idMiddleware()).Post("/posts/{postId}/comments", h.CreateComment)
	r.With(h.idMiddleware()).Patch("/posts/{postId}/comments/{commentId}", h.UpdateComment)
	r.With(h.idMiddleware()).Delete("/posts/{postId}/comments/{commentId}", h.DeleteComment)

	r.With(h.idMiddleware()).Post("/posts/{postId}/reactions", h.CreateReaction)
	r.With(h.idMiddleware()).Delete("/posts/{postId}/reactions/{reactionType}", h.DeleteReaction)

	// Media upload intent.
	r.With(h.idMiddleware()).Post("/media-uploads", h.UploadMedia)
}

// optionalAuthMiddleware returns auth + active persona resolution that allows
// unauthenticated requests to pass through.
func (h *Handler) optionalAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return h.idHandler.OptionalAuthMiddleware(h.idHandler.OptionalActivePersonaMiddleware(next))
	}
}

// idMiddleware returns auth + active persona middleware.
func (h *Handler) idMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return h.idHandler.AuthMiddleware(h.idHandler.ActivePersonaMiddleware(next))
	}
}

// viewerPersonaID returns the active persona ID for an authenticated request,
// or nil for unauthenticated requests. It does not fail if no default persona
// is set, because read endpoints must remain usable.
func (h *Handler) viewerPersonaID(ctx context.Context) *string {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil
	}
	// If active persona middleware ran, use its value.
	if pid := identity.ActivePersonaIDFromContext(ctx); pid != "" {
		return &pid
	}
	profile, err := h.idService.GetMe(ctx, userID)
	if err != nil {
		return nil
	}
	if profile.DefaultPersonaID == nil || *profile.DefaultPersonaID == "" {
		return nil
	}
	return profile.DefaultPersonaID
}

func (h *Handler) requireActivePersona(ctx context.Context, w http.ResponseWriter) (string, bool) {
	pid := identity.ActivePersonaIDFromContext(ctx)
	if pid == "" {
		httpError(ctx, w, http.StatusBadRequest, "PERSONA.DEFAULT_REQUIRED", "please select a default persona first")
		return "", false
	}
	return pid, true
}

func (h *Handler) requireIdempotencyKey(r *http.Request, w http.ResponseWriter) (string, bool) {
	key := r.Header.Get("Idempotency-Key")
	if strings.TrimSpace(key) == "" || len(key) < 8 || len(key) > 128 {
		httpError(r.Context(), w, http.StatusBadRequest, "IDEMPOTENCY.MISSING_KEY", "Idempotency-Key header is required")
		return "", false
	}
	return key, true
}

func (h *Handler) requireUser(ctx context.Context, w http.ResponseWriter) (string, bool) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		httpError(ctx, w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
		return "", false
	}
	return userID, true
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

type topicResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	NoteCount     int    `json:"noteCount"`
	FollowerCount int    `json:"followerCount"`
	IsFollowed    bool   `json:"isFollowed"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
}

type personaResponse struct {
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

type postResponse struct {
	ID              string          `json:"id"`
	Persona         personaResponse `json:"persona"`
	Topic           topicResponse   `json:"topic"`
	Content         string          `json:"content"`
	Media           []any           `json:"media"`
	ReactionCounts  map[string]int  `json:"reactionCounts"`
	UserReaction    *string         `json:"userReaction"`
	IsSaved         bool            `json:"isSaved"`
	ReplyCount      int             `json:"replyCount"`
	ModerationState string          `json:"moderationState"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

type commentResponse struct {
	ID              string          `json:"id"`
	Persona         personaResponse `json:"persona"`
	PostID          string          `json:"postId"`
	Content         string          `json:"content"`
	ReactionCounts  map[string]int  `json:"reactionCounts"`
	UserReaction    *string         `json:"userReaction"`
	ModerationState string          `json:"moderationState"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

type reactionResponse struct {
	ID        string          `json:"id"`
	Persona   personaResponse `json:"persona"`
	Type      string          `json:"type"`
	CreatedAt string          `json:"createdAt"`
}

type reactionSummaryResponse struct {
	ReactionCounts map[string]int `json:"reactionCounts"`
	UserReaction   *string        `json:"userReaction"`
}

type postCreateRequest struct {
	PersonaID *string `json:"personaId"`
	TopicID   string  `json:"topicId"`
	Content   string  `json:"content"`
}

type postUpdateRequest struct {
	TopicID *string `json:"topicId"`
	Content *string `json:"content"`
}

type commentCreateRequest struct {
	PersonaID *string `json:"personaId"`
	Content   string  `json:"content"`
}

type commentUpdateRequest struct {
	Content string `json:"content"`
}

type reactionCreateRequest struct {
	Type string `json:"type"`
}

type mediaUploadIntentRequest struct {
	MimeType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`
	Checksum  string `json:"checksum"`
}

type mediaUploadIntentResponse struct {
	AssetID   string `json:"assetId"`
	UploadURL string `json:"uploadUrl"`
	MimeType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`
	Checksum  string `json:"checksum"`
	Status    string `json:"status"`
}

// ListTopics handles GET /v1/topics.
func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 100 {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "query too long")
		return
	}
	viewer := h.viewerPersonaID(r.Context())
	page, err := h.service.ListTopics(r.Context(), q, r.URL.Query().Get("cursor"), parseLimit(r), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	resp := cursorPageResponse{Data: []any{}, Pagination: paginationResponse{NextCursor: page.NextCursor, HasMore: page.HasMore, Limit: page.Limit}}
	for _, t := range page.Data {
		resp.Data = append(resp.Data, h.toTopicResponse(&t))
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetTopic handles GET /v1/topics/{topicId}.
func (h *Handler) GetTopic(w http.ResponseWriter, r *http.Request) {
	viewer := h.viewerPersonaID(r.Context())
	t, err := h.service.GetTopic(r.Context(), chi.URLParam(r, "topicId"), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toTopicResponse(t)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// FollowTopic handles POST /v1/topics/{topicId}/follow.
func (h *Handler) FollowTopic(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if err := h.service.FollowTopic(r.Context(), userID, personaID, chi.URLParam(r, "topicId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UnfollowTopic handles DELETE /v1/topics/{topicId}/follow.
func (h *Handler) UnfollowTopic(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if err := h.service.UnfollowTopic(r.Context(), userID, personaID, chi.URLParam(r, "topicId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListTopicPosts handles GET /v1/topics/{topicId}/posts.
func (h *Handler) ListTopicPosts(w http.ResponseWriter, r *http.Request) {
	viewer := h.viewerPersonaID(r.Context())
	page, err := h.service.ListTopicPosts(r.Context(), chi.URLParam(r, "topicId"), r.URL.Query().Get("cursor"), parseLimit(r), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	h.writePostPage(w, r, page)
}

// ListPosts handles GET /v1/posts.
func (h *Handler) ListPosts(w http.ResponseWriter, r *http.Request) {
	var topicID *string
	if q := r.URL.Query().Get("topicId"); q != "" {
		topicID = &q
	}
	viewer := h.viewerPersonaID(r.Context())
	page, err := h.service.ListPosts(r.Context(), topicID, r.URL.Query().Get("cursor"), parseLimit(r), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	h.writePostPage(w, r, page)
}

// ListPersonaPosts handles GET /v1/personas/{personaId}/posts.
func (h *Handler) ListPersonaPosts(w http.ResponseWriter, r *http.Request) {
	viewer := h.viewerPersonaID(r.Context())
	page, err := h.service.ListPersonaPosts(r.Context(), chi.URLParam(r, "personaId"), r.URL.Query().Get("cursor"), parseLimit(r), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	h.writePostPage(w, r, page)
}

func (h *Handler) writePostPage(w http.ResponseWriter, r *http.Request, page *CursorPage[PostPublic]) {
	resp := cursorPageResponse{Data: []any{}, Pagination: paginationResponse{NextCursor: page.NextCursor, HasMore: page.HasMore, Limit: page.Limit}}
	for _, p := range page.Data {
		resp.Data = append(resp.Data, h.toPostResponse(&p))
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// CreatePost handles POST /v1/posts.
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	key, ok := h.requireIdempotencyKey(r, w)
	if !ok {
		return
	}

	// Replay a stored response for this idempotency key, if any.
	if cached, err := h.service.GetIdempotencyResponse(r.Context(), key, r.Method, r.URL.Path); err == nil && cached != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(cached.Status)
		_, _ = w.Write(cached.Body)
		return
	}

	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	var req postCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	post, err := h.service.CreatePost(r.Context(), userID, personaID, &PostCreateRequest{
		PersonaID: req.PersonaID,
		TopicID:   req.TopicID,
		Content:   req.Content,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}

	respBody, _ := json.Marshal(h.toPostResponse(post))
	if err := h.service.SaveIdempotencyResponse(r.Context(), key, r.Method, r.URL.Path, http.StatusCreated, respBody); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to record idempotency response")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, h.toPostResponse(post)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetPost handles GET /v1/posts/{postId}.
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	viewer := h.viewerPersonaID(r.Context())
	post, err := h.service.GetPost(r.Context(), chi.URLParam(r, "postId"), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toPostResponse(post)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// UpdatePost handles PATCH /v1/posts/{postId}.
func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	var req postUpdateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	post, err := h.service.UpdatePost(r.Context(), userID, personaID, chi.URLParam(r, "postId"), &PostUpdateRequest{
		TopicID: req.TopicID,
		Content: req.Content,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toPostResponse(post)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// DeletePost handles DELETE /v1/posts/{postId}.
func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if err := h.service.DeletePost(r.Context(), userID, personaID, chi.URLParam(r, "postId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateSave handles POST /v1/posts/{postId}/saves.
func (h *Handler) CreateSave(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if err := h.service.SavePost(r.Context(), userID, personaID, chi.URLParam(r, "postId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSave handles DELETE /v1/posts/{postId}/saves.
func (h *Handler) DeleteSave(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if err := h.service.UnsavePost(r.Context(), userID, personaID, chi.URLParam(r, "postId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListComments handles GET /v1/posts/{postId}/comments.
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	viewer := h.viewerPersonaID(r.Context())
	page, err := h.service.ListComments(r.Context(), chi.URLParam(r, "postId"), r.URL.Query().Get("cursor"), parseLimit(r), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	resp := cursorPageResponse{Data: []any{}, Pagination: paginationResponse{NextCursor: page.NextCursor, HasMore: page.HasMore, Limit: page.Limit}}
	for _, c := range page.Data {
		resp.Data = append(resp.Data, h.toCommentResponse(&c))
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// CreateComment handles POST /v1/posts/{postId}/comments.
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	var req commentCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	comment, err := h.service.CreateComment(r.Context(), userID, personaID, chi.URLParam(r, "postId"), &CommentCreateRequest{
		PersonaID: req.PersonaID,
		Content:   req.Content,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusCreated, h.toCommentResponse(comment)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// GetComment handles GET /v1/posts/{postId}/comments/{commentId}.
func (h *Handler) GetComment(w http.ResponseWriter, r *http.Request) {
	viewer := h.viewerPersonaID(r.Context())
	comment, err := h.service.GetComment(r.Context(), chi.URLParam(r, "commentId"), viewer)
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toCommentResponse(comment)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// UpdateComment handles PATCH /v1/posts/{postId}/comments/{commentId}.
func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	var req commentUpdateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	comment, err := h.service.UpdateComment(r.Context(), userID, personaID, chi.URLParam(r, "commentId"), &CommentUpdateRequest{
		Content: req.Content,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, h.toCommentResponse(comment)); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// DeleteComment handles DELETE /v1/posts/{postId}/comments/{commentId}.
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	if err := h.service.DeleteComment(r.Context(), userID, personaID, chi.URLParam(r, "commentId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListReactions handles GET /v1/posts/{postId}/reactions.
func (h *Handler) ListReactions(w http.ResponseWriter, r *http.Request) {
	page, err := h.service.ListReactions(r.Context(), "post", chi.URLParam(r, "postId"), r.URL.Query().Get("cursor"), parseLimit(r))
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	resp := cursorPageResponse{Data: []any{}, Pagination: paginationResponse{NextCursor: page.NextCursor, HasMore: page.HasMore, Limit: page.Limit}}
	for _, re := range page.Data {
		resp.Data = append(resp.Data, h.toReactionResponse(&re))
	}
	if err := httpx.WriteJSON(w, http.StatusOK, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// CreateReaction handles POST /v1/posts/{postId}/reactions.
func (h *Handler) CreateReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	var req reactionCreateRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}
	summary, err := h.service.CreateReaction(r.Context(), userID, personaID, "post", chi.URLParam(r, "postId"), &ReactionCreateRequest{
		Type: req.Type,
	})
	if err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	if err := httpx.WriteJSON(w, http.StatusOK, reactionSummaryResponse{
		ReactionCounts: summary.ReactionCounts,
		UserReaction:   summary.UserReaction,
	}); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

// DeleteReaction handles DELETE /v1/posts/{postId}/reactions/{reactionType}.
func (h *Handler) DeleteReaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	if _, ok := h.requireIdempotencyKey(r, w); !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}
	reactionType := chi.URLParam(r, "reactionType")
	if reactionType != "like" {
		httpError(r.Context(), w, http.StatusBadRequest, "REACTION.INVALID_TYPE", "that reaction isn't supported")
		return
	}
	if err := h.service.DeleteReaction(r.Context(), userID, personaID, "post", chi.URLParam(r, "postId")); err != nil {
		h.respondDomainError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadMedia handles POST /v1/media-uploads.
func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	_, ok := h.requireUser(r.Context(), w)
	if !ok {
		return
	}
	key, ok := h.requireIdempotencyKey(r, w)
	if !ok {
		return
	}
	personaID, ok := h.requireActivePersona(r.Context(), w)
	if !ok {
		return
	}

	var req mediaUploadIntentRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid request body")
		return
	}

	const maxUploadSize = 10 * 1024 * 1024
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowedTypes[strings.ToLower(req.MimeType)] {
		httpError(r.Context(), w, http.StatusUnprocessableEntity, "MEDIA.INVALID_FORMAT", "unsupported media format")
		return
	}
	if req.SizeBytes <= 0 || req.SizeBytes > maxUploadSize {
		httpError(r.Context(), w, http.StatusUnprocessableEntity, "MEDIA.TOO_LARGE", "media size is not allowed")
		return
	}
	if strings.TrimSpace(req.Checksum) == "" || len(req.Checksum) > 128 {
		httpError(r.Context(), w, http.StatusBadRequest, "VALIDATION_FAILED", "checksum is required")
		return
	}

	asset := &MediaAsset{
		PersonaID: personaID,
		URL:       fmt.Sprintf("%s://%s/v1/media-uploads/%s/blob", requestScheme(r), r.Host, ""),
		MimeType:  req.MimeType,
		Width:     0,
		Height:    0,
		FileSize:  &req.SizeBytes,
		Checksum:  &req.Checksum,
		Status:    "pending",
	}
	if err := h.service.CreateMediaAsset(r.Context(), asset); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create media intent")
		return
	}

	// Replace placeholder ID now that the asset has been created.
	asset.URL = fmt.Sprintf("%s://%s/v1/media-uploads/%s/blob", requestScheme(r), r.Host, asset.ID)

	resp := mediaUploadIntentResponse{
		AssetID:   asset.ID,
		UploadURL: asset.URL,
		MimeType:  asset.MimeType,
		SizeBytes: req.SizeBytes,
		Checksum:  req.Checksum,
		Status:    asset.Status,
	}

	// Store idempotency response so retries replay the same intent.
	respBody, _ := json.Marshal(resp)
	if err := h.service.SaveIdempotencyResponse(r.Context(), key, r.Method, r.URL.Path, http.StatusCreated, respBody); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to record idempotency response")
		return
	}

	if err := httpx.WriteJSON(w, http.StatusCreated, resp); err != nil {
		httpError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to write response")
	}
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return "https"
	}
	return "http"
}

func (h *Handler) toTopicResponse(t *TopicPublic) topicResponse {
	return topicResponse{
		ID:            t.ID,
		Name:          t.Name,
		Description:   t.Description,
		Category:      t.Category,
		NoteCount:     t.NoteCount,
		FollowerCount: t.FollowerCount,
		IsFollowed:    t.IsFollowed,
		Status:        t.Status,
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
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
		IsBlocked: false,
	}
}

func (h *Handler) toPostResponse(p *PostPublic) postResponse {
	resp := postResponse{
		ID:              p.ID,
		Persona:         h.toPersonaResponse(p.Persona),
		Topic:           h.toTopicResponse(p.Topic),
		Content:         p.Content,
		Media:           p.Media,
		ReactionCounts:  p.ReactionCounts,
		UserReaction:    p.UserReaction,
		IsSaved:         p.IsSaved,
		ReplyCount:      p.ReplyCount,
		ModerationState: p.ModerationState,
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
	}
	if resp.ReactionCounts == nil {
		resp.ReactionCounts = map[string]int{}
	}
	if resp.Media == nil {
		resp.Media = []any{}
	}
	return resp
}

func (h *Handler) toCommentResponse(c *CommentPublic) commentResponse {
	resp := commentResponse{
		ID:              c.ID,
		Persona:         h.toPersonaResponse(c.Persona),
		PostID:          c.PostID,
		Content:         c.Content,
		ReactionCounts:  c.ReactionCounts,
		UserReaction:    c.UserReaction,
		ModerationState: c.ModerationState,
		CreatedAt:       c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       c.UpdatedAt.Format(time.RFC3339),
	}
	if resp.ReactionCounts == nil {
		resp.ReactionCounts = map[string]int{}
	}
	return resp
}

func (h *Handler) toReactionResponse(re *ReactionPublic) reactionResponse {
	return reactionResponse{
		ID:        re.ID,
		Persona:   h.toPersonaResponse(re.Persona),
		Type:      re.Type,
		CreatedAt: re.CreatedAt.Format(time.RFC3339),
	}
}

func (h *Handler) respondDomainError(ctx context.Context, w http.ResponseWriter, err error) {
	var rateLimitErr *auth.RateLimitError
	switch {
	case errors.Is(err, identity.ErrPersonaNotFound), errors.Is(err, identity.ErrPersonaRestricted):
		httpError(ctx, w, http.StatusForbidden, "PERSONA.RESTRICTED", "persona cannot be used")
	case errors.Is(err, ErrTopicNotFound):
		httpError(ctx, w, http.StatusNotFound, "TOPIC.NOT_FOUND", "that channel doesn't exist")
	case errors.Is(err, ErrTopicAlreadyFollowed):
		httpError(ctx, w, http.StatusConflict, "TOPIC.ALREADY_FOLLOWED", "you're already following this channel")
	case errors.Is(err, ErrTopicNotFollowed):
		httpError(ctx, w, http.StatusConflict, "TOPIC.NOT_FOLLOWED", "you aren't following this channel")
	case errors.Is(err, ErrTopicHidden):
		httpError(ctx, w, http.StatusForbidden, "TOPIC.HIDDEN", "this channel isn't available")
	case errors.Is(err, ErrPostNotFound):
		httpError(ctx, w, http.StatusNotFound, "POST.NOT_FOUND", "that note isn't available")
	case errors.Is(err, ErrPostNotAuthor):
		httpError(ctx, w, http.StatusForbidden, "POST.NOT_AUTHOR", "you can only edit your own notes")
	case errors.Is(err, ErrPostInvalidState):
		httpError(ctx, w, http.StatusConflict, "POST.INVALID_STATE", "this note can't be edited right now")
	case errors.Is(err, ErrPostContentDisallowed):
		httpError(ctx, w, http.StatusUnprocessableEntity, "POST.CONTENT_DISALLOWED", "this note couldn't be published due to safety guidelines")
	case errors.Is(err, ErrPostTopicRequired):
		httpError(ctx, w, http.StatusUnprocessableEntity, "POST.TOPIC_REQUIRED", "please choose a channel")
	case errors.As(err, &rateLimitErr):
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rateLimitErr.RetryAfter.Seconds())))
		httpError(ctx, w, http.StatusTooManyRequests, "POST.RATE_LIMITED", "you're posting too quickly. please slow down")
	case errors.Is(err, ErrCommentNotFound):
		httpError(ctx, w, http.StatusNotFound, "COMMENT.NOT_FOUND", "that reply isn't available")
	case errors.Is(err, ErrCommentNotAuthor):
		httpError(ctx, w, http.StatusForbidden, "COMMENT.NOT_AUTHOR", "you can only edit your own replies")
	case errors.Is(err, ErrCommentInvalidState):
		httpError(ctx, w, http.StatusConflict, "COMMENT.INVALID_STATE", "this reply can't be edited right now")
	case errors.Is(err, ErrCommentContentDisallowed):
		httpError(ctx, w, http.StatusUnprocessableEntity, "COMMENT.CONTENT_DISALLOWED", "this reply couldn't be posted due to safety guidelines")
	case errors.Is(err, ErrCommentParentNotFound):
		httpError(ctx, w, http.StatusNotFound, "COMMENT.PARENT_NOT_FOUND", "the note you're replying to isn't available")
	case errors.Is(err, ErrCommentRateLimited):
		httpError(ctx, w, http.StatusTooManyRequests, "COMMENT.RATE_LIMITED", "you're replying too quickly. please slow down")
	case errors.Is(err, ErrReactionInvalidType):
		httpError(ctx, w, http.StatusBadRequest, "REACTION.INVALID_TYPE", "that reaction isn't supported")
	case errors.Is(err, ErrReactionAlreadyExists):
		httpError(ctx, w, http.StatusConflict, "REACTION.ALREADY_EXISTS", "you've already reacted to this")
	case errors.Is(err, ErrReactionNotFound):
		httpError(ctx, w, http.StatusNotFound, "REACTION.NOT_FOUND", "reaction not found")
	case errors.Is(err, ErrReactionTargetNotFound):
		httpError(ctx, w, http.StatusNotFound, "REACTION.TARGET_NOT_FOUND", "that note or reply isn't available")
	case errors.Is(err, ErrSaveAlreadyExists):
		httpError(ctx, w, http.StatusConflict, "SAVE.ALREADY_EXISTS", "you've already saved this note")
	case errors.Is(err, ErrSaveNotFound):
		httpError(ctx, w, http.StatusNotFound, "SAVE.NOT_FOUND", "you haven't saved this note")
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
