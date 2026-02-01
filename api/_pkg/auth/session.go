package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
)

type AuthenticatedHandler func(w http.ResponseWriter, r *http.Request, playerID uuid.UUID)

func RequireAuth(next AuthenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID, err := AuthenticateRequest(r)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized: "+err.Error())
			return
		}
		next(w, r, playerID)
	}
}

func GenerateSessionToken() (plaintext string, hash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	plaintext = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	hash = HashToken(plaintext)
	return plaintext, hash, nil
}

func HashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

func AuthenticateRequest(r *http.Request) (uuid.UUID, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return uuid.Nil, fmt.Errorf("missing Authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return uuid.Nil, fmt.Errorf("invalid Authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return uuid.Nil, fmt.Errorf("empty bearer token")
	}

	tokenHash := HashToken(token)

	database, err := db.GetDB()
	if err != nil {
		return uuid.Nil, fmt.Errorf("database error: %w", err)
	}

	session, err := db.FindSessionByTokenHash(database, tokenHash)
	if err != nil {
		return uuid.Nil, fmt.Errorf("session lookup error: %w", err)
	}
	if session == nil {
		return uuid.Nil, fmt.Errorf("invalid session token")
	}

	if time.Now().After(session.ExpiresAt) {
		return uuid.Nil, fmt.Errorf("session token has expired")
	}

	return session.PlayerID, nil
}
