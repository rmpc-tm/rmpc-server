package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"rmpc-server/api/_pkg/auth"
	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
	"rmpc-server/api/_pkg/validate"
)

type scoreSubmitRequest struct {
	GameMode      string          `json:"game_mode"      validate:"required,oneof=author gold custom"`
	Score         int32           `json:"score"           validate:"gte=0"`
	MapsCompleted int32           `json:"maps_completed"  validate:"gte=0"`
	MapsSkipped   int32           `json:"maps_skipped"    validate:"gte=0"`
	DurationMs    int32           `json:"duration_ms"     validate:"gte=60000,lte=7200000"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type scoreSubmitResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func Scores(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	auth.RequireAuth(handleScoreSubmit)(w, r)
}

func handleScoreSubmit(w http.ResponseWriter, r *http.Request, playerID uuid.UUID) {
	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Check ban
	banned, err := db.IsPlayerBanned(database, playerID)
	if err != nil {
		slog.Error("ban check error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	if banned {
		// Fake OK
		response.JSON(w, http.StatusCreated, scoreSubmitResponse{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
		})
		return
	}

	// Parse request
	var req scoreSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	// Validate metadata
	if len(req.Metadata) > 0 {
		if len(req.Metadata) > 256*1024 {
			response.Error(w, http.StatusBadRequest, "metadata must not exceed 256KB")
			return
		}
		var metaMap map[string]interface{}
		if err := json.Unmarshal(req.Metadata, &metaMap); err != nil {
			response.Error(w, http.StatusBadRequest, "metadata must be a JSON object")
			return
		}
		if len(metaMap) > 10 {
			response.Error(w, http.StatusBadRequest, "metadata must not have more than 10 keys")
			return
		}
	}

	// Check cooldown
	canSubmit, err := db.CanSubmitScore(database, playerID, config.Env.ScoreCooldown)
	if err != nil {
		slog.Error("cooldown check error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	if !canSubmit {
		response.Error(w, http.StatusTooManyRequests, "please wait before submitting another score")
		return
	}

	// Convert metadata to *string for go-jet JSONB column
	var metadata *string
	if len(req.Metadata) > 0 {
		s := string(req.Metadata)
		metadata = &s
	}

	// Insert score
	id, createdAt, err := db.InsertScore(database, db.ScoreInput{
		PlayerID:      playerID,
		GameMode:      req.GameMode,
		Score:         req.Score,
		MapsCompleted: req.MapsCompleted,
		MapsSkipped:   req.MapsSkipped,
		DurationMs:    req.DurationMs,
		Metadata:      metadata,
	})
	if err != nil {
		slog.Error("insert score error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	response.JSON(w, http.StatusCreated, scoreSubmitResponse{
		ID:        id.String(),
		CreatedAt: createdAt,
	})
}
