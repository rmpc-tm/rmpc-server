package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"rmpc-server/api/_pkg/config"
)

var openplanetClient = &http.Client{
	Timeout: 10 * time.Second,
}

var (
	// ErrInvalidToken: Openplanet answered and rejected the token. Reauth.
	ErrInvalidToken = errors.New("openplanet: token rejected")

	// ErrUpstream: we could not get an answer out of Openplanet at all. Retry.
	ErrUpstream = errors.New("openplanet: validation service unavailable")

	// ErrMisconfigured: internal.
	ErrMisconfigured = errors.New("openplanet: server is misconfigured")
)

type OpenplanetUser struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
}

type openplanetValidateRequest struct {
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

// openplanetValidateResponse covers both shapes: a success carries account_id,
// a rejection carries a human-readable error.
type openplanetValidateResponse struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
	Error       string `json:"error"`
}

func ValidateOpenplanetToken(token string) (*OpenplanetUser, error) {
	secret := config.Env.OpenplanetPluginSecret
	if secret == "" {
		return nil, fmt.Errorf("%w: OPENPLANET_PLUGIN_SECRET is not set", ErrMisconfigured)
	}

	body, err := json.Marshal(openplanetValidateRequest{
		Token:  token,
		Secret: secret,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal request: %v", ErrMisconfigured, err)
	}

	resp, err := openplanetClient.Post(
		config.Env.OpenplanetAuthURL,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to connect to Openplanet API: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // 64KB max
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read Openplanet response: %v", ErrUpstream, err)
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		// continue
	case resp.StatusCode >= 500:
		return nil, fmt.Errorf("%w: Openplanet returned status %d: %s",
			ErrUpstream, resp.StatusCode, preview(string(respBody)))
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: Openplanet returned status %d: %s",
			ErrUpstream, resp.StatusCode, preview(string(respBody)))
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("%w: Openplanet returned status %d: %s",
			ErrInvalidToken, resp.StatusCode, preview(string(respBody)))
	default:
		// 1xx/3xx
		return nil, fmt.Errorf("%w: Openplanet returned unexpected status %d",
			ErrUpstream, resp.StatusCode)
	}

	var parsed openplanetValidateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("%w: failed to parse Openplanet response: %v", ErrUpstream, err)
	}

	if parsed.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, preview(parsed.Error))
	}

	if parsed.AccountID == "" {
		return nil, fmt.Errorf("%w: Openplanet returned no account_id", ErrInvalidToken)
	}

	return &OpenplanetUser{
		AccountID:   parsed.AccountID,
		DisplayName: parsed.DisplayName,
	}, nil
}

// preview trims an upstream response down to something loggable.
func preview(s string) string {
	pLen := 120
	s = strings.TrimSpace(s)
	if len(s) <= pLen {
		return s
	}
	return s[:pLen] + "..."
}

func GetClientIP(r *http.Request) string {
	// Prefer X-Real-Ip set by trusted reverse proxies (Vercel, nginx)
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	// Fall back to rightmost X-Forwarded-For entry (appended by the proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
