// Package playerlink signs and verifies short tokens for per-player page URLs.
//
// The token is the first SigLen base64url chars of HMAC-SHA256(secret, openplanetID).
// Without the secret an attacker cannot mint tokens for arbitrary IDs, so the
// /api/player handler can short-circuit before touching the database.
package playerlink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"rmpc-server/api/_pkg/config"
)

// SigLen is the length of the truncated base64url HMAC (~96 bits — unforgeable, short).
const SigLen = 16

func Sign(openplanetID string) string {
	secret := config.Env.PlayerLinkSecret
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(openplanetID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:SigLen]
}

func Verify(openplanetID, sig string) bool {
	if len(sig) != SigLen {
		return false
	}
	expected := Sign(openplanetID)
	if expected == "" {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(sig))
}
