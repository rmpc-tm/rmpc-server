package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"rmpc-server/api/_pkg/config"
)

type OpenplanetUser struct {
	AccountID   string `json:"account_id"`
	DisplayName string `json:"display_name"`
}

type openplanetValidateRequest struct {
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

func ValidateOpenplanetToken(token string) (*OpenplanetUser, error) {
	secret := config.Env.OpenplanetPluginSecret
	if secret == "" {
		return nil, fmt.Errorf("OPENPLANET_PLUGIN_SECRET is not set")
	}

	body, err := json.Marshal(openplanetValidateRequest{
		Token:  token,
		Secret: secret,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(
		config.Env.OpenplanetAuthURL,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to contact Openplanet API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Openplanet response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openplanet validation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var user OpenplanetUser
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse Openplanet response: %w", err)
	}

	if user.AccountID == "" {
		return nil, fmt.Errorf("openplanet returned empty account_id")
	}

	return &user, nil
}

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client IP)
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
