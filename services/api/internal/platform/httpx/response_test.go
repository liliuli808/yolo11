package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorResponse_WritesExpectedShape(t *testing.T) {
	ctx := context.WithValue(context.Background(), requestIDKey, "req-123")

	rec := httptest.NewRecorder()
	Error(ctx, rec, http.StatusBadRequest, "INVALID_INPUT", "username is required")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content type, got %q", ct)
	}

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body.Code != "INVALID_INPUT" {
		t.Errorf("expected code INVALID_INPUT, got %q", body.Code)
	}
	if body.Message != "username is required" {
		t.Errorf("expected message username is required, got %q", body.Message)
	}
	if body.RequestID != "req-123" {
		t.Errorf("expected requestId req-123, got %q", body.RequestID)
	}
}

func TestErrorResponse_DefaultRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(context.Background(), rec, http.StatusInternalServerError, "INTERNAL_ERROR", "something went wrong")

	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if body.RequestID == "" {
		t.Error("expected a non-empty requestId when none is in context")
	}
}

func TestWriteJSON_Success(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"status": "ok"}

	if err := WriteJSON(rec, http.StatusOK, payload); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected JSON content type, got %q", rec.Header().Get("Content-Type"))
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %q", body["status"])
	}
}

func TestWriteJSON_EncodeError(t *testing.T) {
	rec := httptest.NewRecorder()
	// Channels cannot be JSON encoded.
	badPayload := make(chan int)

	err := WriteJSON(rec, http.StatusOK, badPayload)
	if err == nil {
		t.Fatal("expected an error for unencodable payload")
	}
}

func TestReadJSON_DecodesBody(t *testing.T) {
	payload := map[string]string{"name": "test"}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))

	var dest map[string]string
	if err := ReadJSON(req, &dest); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}
	if dest["name"] != "test" {
		t.Errorf("expected name test, got %q", dest["name"])
	}
}

func TestReadJSON_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not json")))

	var dest map[string]string
	err := ReadJSON(req, &dest)
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestReadJSON_BodyTooLarge(t *testing.T) {
	payload := map[string]string{"data": strings.Repeat("x", 1024)}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))

	var dest map[string]string
	err := ReadJSONWithLimit(req, &dest, 100)
	if err == nil {
		t.Fatal("expected an error for oversized request body")
	}
}
