package handler

import (
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
	"rmpc-server/api/_pkg/validate"
)

type leaderboardQuery struct {
	GameMode string `json:"game_mode" validate:"omitempty,oneof=author gold"`
	Period   string `json:"period"    validate:"oneof=all year month past_month"`
}

type leaderboardResponse struct {
	Scores   []leaderboardEntryJSON `json:"scores"`
	Period   string                 `json:"period"`
	GameMode string                 `json:"game_mode"`
}

type leaderboardPlayerJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type leaderboardEntryJSON struct {
	Rank          int                   `json:"rank"`
	Player        leaderboardPlayerJSON `json:"player"`
	Score         int32                 `json:"score"`
	MapsCompleted int32                 `json:"maps_completed"`
	MapsSkipped   int32                 `json:"maps_skipped"`
	DurationMs    int32                 `json:"duration_ms"`
	GameMode      string                `json:"game_mode"`
	CreatedAt     time.Time             `json:"created_at"`
}

func Leaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()

	query := leaderboardQuery{
		GameMode: q.Get("game_mode"),
		Period:   q.Get("period"),
	}
	if query.Period == "" {
		query.Period = "all"
	}
	if err := validate.Struct(query); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	var startTime *time.Time
	now := time.Now()
	switch query.Period {
	case "year":
		t := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		startTime = &t
	case "month":
		t := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		startTime = &t
	case "past_month":
		t := now.AddDate(0, -1, 0)
		startTime = &t
	}

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	entries, err := db.GetLeaderboard(database, db.LeaderboardParams{
		GameMode:  query.GameMode,
		StartTime: startTime,
	})
	if err != nil {
		slog.Error("leaderboard query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	scores := make([]leaderboardEntryJSON, len(entries))
	for i, e := range entries {
		createdAt := time.Time{}
		if e.CreatedAt != nil {
			createdAt = *e.CreatedAt
		}
		scores[i] = leaderboardEntryJSON{
			Rank: e.Rank,
			Player: leaderboardPlayerJSON{
				ID:          e.PlayerID.String(),
				DisplayName: e.DisplayName,
			},
			Score:         e.Score,
			MapsCompleted: e.MapsCompleted,
			MapsSkipped:   e.MapsSkipped,
			DurationMs:    e.DurationMs,
			GameMode:      e.GameMode.String(),
			CreatedAt:     createdAt,
		}
	}

	gameModeResp := query.GameMode
	if gameModeResp == "" {
		gameModeResp = "all"
	}

	response.JSON(w, http.StatusOK, leaderboardResponse{
		Scores:   scores,
		Period:   query.Period,
		GameMode: gameModeResp,
	})
}
