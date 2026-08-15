package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDHeader is the HTTP header used to carry or return a request ID.
const RequestIDHeader = "X-Request-ID"

// DefaultMaxBodySize is the maximum JSON request body size ReadJSON will
// accept unless a custom limit is supplied.
const DefaultMaxBodySize = 1 << 20 // 1 MiB

type requestIDCtxKey struct{}

var requestIDKey = requestIDCtxKey{}

// errorResponse is the standardized error envelope returned by all API errors.
type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// Error writes a standardized JSON error response to w using the request ID
// stored in ctx. If no request ID is present, a new one is generated.
func Error(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	reqID := RequestIDFromContext(ctx)
	if reqID == "" {
		reqID = uuid.NewString()
	}

	resp := errorResponse{
		Code:      code,
		Message:   message,
		RequestID: reqID,
	}

	w.Header().Set(RequestIDHeader, reqID)
	// Response writing is best-effort; if it fails there is no way to report
	// the failure back to the client on this connection.
	_ = WriteJSON(w, status, resp)
}

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

// ReadJSON parses a JSON request body into dest. It enforces a maximum body
// size of DefaultMaxBodySize to protect against large or streaming payloads.
func ReadJSON(r *http.Request, dest any) error {
	return ReadJSONWithLimit(r, dest, DefaultMaxBodySize)
}

// ReadJSONWithLimit parses a JSON request body into dest, limiting the body to
// maxBytes. A value of zero or negative disables the limit.
func ReadJSONWithLimit(r *http.Request, dest any, maxBytes int64) error {
	var body []byte
	var err error

	if maxBytes > 0 {
		body, err = io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
		if err != nil {
			return fmt.Errorf("read request body: %w", err)
		}
		if int64(len(body)) > maxBytes {
			return fmt.Errorf("request body exceeds maximum allowed size of %d bytes", maxBytes)
		}
	} else {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("read request body: %w", err)
		}
	}

	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	return nil
}

// RequestIDFromContext returns the request ID stored in ctx, if any.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// ContextWithRequestID returns a new context carrying the given request ID.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
