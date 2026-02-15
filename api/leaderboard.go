package handler

import (
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
	"rmpc-server/api/_pkg/validate"
)

type leaderboardQuery struct {
	GameMode string `json:"game_mode" validate:"omitempty,oneof=author gold"`
	Month    string `json:"month"     validate:"omitempty"`
}

type leaderboardResponse struct {
	Scores   []leaderboardEntryJSON `json:"scores"`
	Month    string                 `json:"month,omitempty"`
	GameMode string                 `json:"game_mode"`
}

type leaderboardPlayerJSON struct {
	OpenplanetID string `json:"openplanet_id"`
	DisplayName  string `json:"display_name"`
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

func writeLeaderboardResponse(w http.ResponseWriter, scores []leaderboardEntryJSON, month, gameMode string) {
	if gameMode == "" {
		gameMode = "all"
	}
	response.SetCache(w, config.Env.LeaderboardCacheTTL)
	response.JSON(w, http.StatusOK, leaderboardResponse{
		Scores:   scores,
		Month:    month,
		GameMode: gameMode,
	})
}

// No leaderboard data exists before this month.
var leaderboardEarliestMonth = time.Date(2025, time.November, 1, 0, 0, 0, 0, time.UTC)

func Leaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()

	query := leaderboardQuery{
		GameMode: q.Get("game_mode"),
		Month:    q.Get("month"),
	}
	if err := validate.Struct(query); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	var startTime *time.Time
	var endTime *time.Time
	if query.Month != "" {
		t, err := time.Parse("2006-01", query.Month)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid month format, expected YYYY-MM")
			return
		}

		// return empty leaderboard for requests outside expected range
		now := time.Now().UTC()
		currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		if t.Before(leaderboardEarliestMonth) || t.After(currentMonth) {
			writeLeaderboardResponse(w, []leaderboardEntryJSON{}, query.Month, query.GameMode)
			return
		}

		end := t.AddDate(0, 1, 0)
		startTime = &t
		endTime = &end
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
		EndTime:   endTime,
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
				OpenplanetID: e.OpenplanetID,
				DisplayName:  e.DisplayName,
			},
			Score:         e.Score,
			MapsCompleted: e.MapsCompleted,
			MapsSkipped:   e.MapsSkipped,
			DurationMs:    e.DurationMs,
			GameMode:      e.GameMode.String(),
			CreatedAt:     createdAt,
		}
	}

	writeLeaderboardResponse(w, scores, query.Month, query.GameMode)
}
