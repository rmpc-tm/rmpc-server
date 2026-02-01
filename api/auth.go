package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/auth"
	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/ratelimit"
	"rmpc-server/api/_pkg/response"
	"rmpc-server/api/_pkg/validate"
)

type authRequest struct {
	OpenplanetToken string `json:"openplanet_token" validate:"required"`
}

type authResponse struct {
	SessionToken string    `json:"session_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Rate limit by IP
	ip := auth.GetClientIP(r)
	if !ratelimit.AuthLimiter().Allow(ip) {
		response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	// Parse request
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	// Validate with Openplanet
	user, err := auth.ValidateOpenplanetToken(req.OpenplanetToken)
	if err != nil {
		slog.Error("openplanet validation error", "error", err)
		response.Error(w, http.StatusUnauthorized, "invalid openplanet token")
		return
	}

	// Get DB
	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Upsert player
	playerID, err := db.UpsertPlayer(database, user.AccountID, user.DisplayName)
	if err != nil {
		slog.Error("upsert player error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Generate session token
	plaintext, hash, err := auth.GenerateSessionToken()
	if err != nil {
		slog.Error("token generation error", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Determine expiry
	expiresAt := time.Now().Add(config.Env.SessionTokenExpiry)

	// Store session
	if err := db.CreateSession(database, playerID, hash, expiresAt); err != nil {
		slog.Error("create session error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	response.JSON(w, http.StatusOK, authResponse{
		SessionToken: plaintext,
		ExpiresAt:    expiresAt,
	})
}
