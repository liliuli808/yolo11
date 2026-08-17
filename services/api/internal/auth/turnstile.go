package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yiguan/api/internal/platform/config"
)

// TurnstileVerifier validates a Cloudflare Turnstile token.
type TurnstileVerifier interface {
	Verify(ctx context.Context, token string) error
}

const siteverifyProductionURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// CloudflareTurnstileVerifier verifies tokens against the Siteverify API.
type CloudflareTurnstileVerifier struct {
	secretKey     string
	hostname      string
	client        *http.Client
	siteverifyURL string
}

// NewCloudflareTurnstileVerifier creates a verifier from configuration.
func NewCloudflareTurnstileVerifier(cfg *config.Config) *CloudflareTurnstileVerifier {
	return &CloudflareTurnstileVerifier{
		secretKey:     cfg.TurnstileSecretKey,
		hostname:      cfg.TurnstileExpectedHostname,
		client:        &http.Client{Timeout: 5 * time.Second},
		siteverifyURL: siteverifyProductionURL,
	}
}

// Verify validates token. Empty tokens always fail.
func (v *CloudflareTurnstileVerifier) Verify(ctx context.Context, token string) error {
	if token == "" {
		return ErrCaptchaFailed
	}
	payload, err := json.Marshal(map[string]string{
		"secret":   v.secretKey,
		"response": token,
	})
	if err != nil {
		return fmt.Errorf("encode siteverify payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		v.siteverifyURL,
		bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("call siteverify: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read siteverify response: %w", err)
	}

	var result struct {
		Success    bool     `json:"success"`
		Hostname   string   `json:"hostname"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode siteverify response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("%w: %v", ErrCaptchaFailed, result.ErrorCodes)
	}
	if v.hostname != "" && result.Hostname != v.hostname {
		return ErrCaptchaFailed
	}
	return nil
}

// StubTurnstile is a test double that always passes unless Fail is set.
type StubTurnstile struct {
	Fail bool
}

// Verify implements TurnstileVerifier.
func (s *StubTurnstile) Verify(ctx context.Context, token string) error {
	if s.Fail {
		return ErrCaptchaFailed
	}
	return nil
}
