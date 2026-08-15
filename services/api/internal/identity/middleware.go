package identity

import (
	"context"
	"net/http"

	"github.com/yiguan/api/internal/auth"
	"github.com/yiguan/api/internal/platform/httpx"
)

type personaIDKey int

const activePersonaIDKey personaIDKey = iota

// ActivePersonaMiddleware resolves the authenticated user's default persona and
// stores the active persona ID in request context. Content handlers read the
// persona ID from context. If the user has no active/default persona, it
// responds with PERSONA.DEFAULT_REQUIRED.
func (h *Handler) ActivePersonaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			httpx.Error(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
			return
		}

		profile, err := h.service.GetMe(r.Context(), userID)
		if err != nil {
			if err == ErrProfileNotFound {
				httpx.Error(r.Context(), w, http.StatusNotFound, "ME.NOT_FOUND", "account not found")
				return
			}
			httpx.Error(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
			return
		}

		if profile.DefaultPersonaID == nil || *profile.DefaultPersonaID == "" {
			httpx.Error(r.Context(), w, http.StatusBadRequest, "PERSONA.DEFAULT_REQUIRED", "please select a default persona first")
			return
		}

		p, err := h.service.GetPrivatePersona(r.Context(), userID, *profile.DefaultPersonaID)
		if err != nil {
			if err == ErrPersonaNotFound {
				httpx.Error(r.Context(), w, http.StatusBadRequest, "PERSONA.DEFAULT_REQUIRED", "please select a default persona first")
				return
			}
			httpx.Error(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong. please try again")
			return
		}
		if p.Status != "active" {
			httpx.Error(r.Context(), w, http.StatusForbidden, "PERSONA.RESTRICTED", "persona cannot be used")
			return
		}

		ctx := context.WithValue(r.Context(), activePersonaIDKey, p.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalActivePersonaMiddleware resolves the authenticated user's default
// persona when possible and stores the active persona ID in the request context.
// Unlike ActivePersonaMiddleware, it never fails: if there is no user, no default
// persona, or the default persona is unusable, it simply continues without
// setting an active persona so public read endpoints remain available.
func (h *Handler) OptionalActivePersonaMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserIDFromContext(r.Context())
		if userID == "" {
			next.ServeHTTP(w, r)
			return
		}

		profile, err := h.service.GetMe(r.Context(), userID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		if profile.DefaultPersonaID == nil || *profile.DefaultPersonaID == "" {
			next.ServeHTTP(w, r)
			return
		}

		p, err := h.service.GetPrivatePersona(r.Context(), userID, *profile.DefaultPersonaID)
		if err != nil || p.Status != "active" {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), activePersonaIDKey, p.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ActivePersonaIDFromContext returns the active persona ID placed by
// ActivePersonaMiddleware, or empty string if none.
func ActivePersonaIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(activePersonaIDKey).(string)
	return v
}
