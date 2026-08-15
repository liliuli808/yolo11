package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

// RateLimiter is the interface implemented by per-route rate limiters.
type RateLimiter interface {
	// Allow returns true if the request identified by key is permitted to
	// proceed. When false, retryAfter indicates how long the client should wait.
	Allow(ctx context.Context, key string) (bool, time.Duration, error)
}

// CORSConfig configures the CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string
	Environment    string
}

// RateLimitConfig configures per-route rate limiting.
type RateLimitConfig struct {
	BehindProxy bool
}

// RequestID adds a unique request ID to the request context and response header.
// If the incoming request already carries the header, that value is reused.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		w.Header().Set(RequestIDHeader, id)
		r = r.WithContext(ContextWithRequestID(r.Context(), id))
		next.ServeHTTP(w, r)
	})
}

// Logger returns a middleware that logs request details with structured logging.
// When behindProxy is true the logged client IP is derived from the same
// headers and fallback logic used by the rate limiter.
func Logger(logger *slog.Logger, behindProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", RequestIDFromContext(r.Context())),
				slog.String("client_ip", clientIP(r, behindProxy)),
			)
		})
	}
}

// Recovery returns a middleware that recovers from panics, logs them, and
// returns a standardized 500 error response.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						slog.String("request_id", RequestIDFromContext(r.Context())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						slog.Any("error", rec),
					)
					Error(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS returns a CORS middleware with sensible defaults for an API.
// In development, an empty AllowedOrigins list defaults to "*". In other
// environments the caller must supply explicit origins.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 && cfg.Environment == "development" {
		allowedOrigins = []string{"*"}
	}
	return cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "Idempotency-Key"},
		ExposedHeaders:   []string{"X-Request-ID", "Idempotency-Key"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}

// RateLimit returns a per-route rate-limiting middleware using the provided
// limiter. The key is derived from the request method, path, and client IP.
// When cfg.BehindProxy is true, the client IP is read from the rightmost entry
// in X-Forwarded-For (the address added by the trusted immediate upstream
// proxy), falling back to X-Real-IP and finally RemoteAddr; otherwise the port
// is stripped from RemoteAddr.
func RateLimit(limiter RateLimiter, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := fmt.Sprintf("%s:%s:%s", r.Method, r.URL.Path, clientIP(r, cfg.BehindProxy))
			allowed, retryAfter, err := limiter.Allow(r.Context(), key)
			if err != nil {
				Error(r.Context(), w, http.StatusInternalServerError, "RATE_LIMIT_ERROR", "rate limiter unavailable")
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
				Error(r.Context(), w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request, behindProxy bool) string {
	if behindProxy {
		// X-Forwarded-For is a list of addresses appended to by each proxy as a
		// request is forwarded. The leftmost entries are client-controlled and
		// trivially forged, so use the rightmost entry added by the trusted
		// immediate upstream proxy, then fall back to X-Real-IP, and finally
		// RemoteAddr. Empty or whitespace-only entries are ignored.
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
