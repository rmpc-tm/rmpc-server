package handler

import (
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/playerlink"
	"rmpc-server/api/_pkg/response"
	"rmpc-server/api/_pkg/validate"
)

type playerQuery struct {
	ID  string `json:"id"  validate:"required"`
	Sig string `json:"t"   validate:"required"`
}

type playerScoreJSON struct {
	Score         int32     `json:"score"`
	MapsCompleted int32     `json:"maps_completed"`
	MapsSkipped   int32     `json:"maps_skipped"`
	DurationMs    int32     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}

type playerModeJSON struct {
	GameMode  string            `json:"game_mode"`
	BestScore int32             `json:"best_score"`
	RunCount  int               `json:"run_count"`
	Scores    []playerScoreJSON `json:"scores"`
}

type playerSummaryJSON struct {
	TotalRuns    int    `json:"total_runs"`
	MedalsEarned int64  `json:"medals_earned"`
	MapsSkipped  int64  `json:"maps_skipped"`
	ActiveSince  string `json:"active_since"`
}

type playerHeaderJSON struct {
	OpenplanetID string `json:"openplanet_id"`
	DisplayName  string `json:"display_name"`
}

type playerResponse struct {
	Player  playerHeaderJSON  `json:"player"`
	Summary playerSummaryJSON `json:"summary"`
	Modes   []playerModeJSON  `json:"modes"`
}

func Player(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()
	query := playerQuery{ID: q.Get("id"), Sig: q.Get("t")}
	if err := validate.Struct(query); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	// Reject before DB. 404s are cached so repeat bad-link traffic doesn't
	// re-invoke this function on every request.
	w.Header().Set("X-Robots-Tag", "noindex")
	if !playerlink.Verify(query.ID, query.Sig) {
		response.SetCache(w, config.Env.PlayerCacheTTL)
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	detail, err := db.GetPlayerDetail(database, query.ID)
	if err != nil {
		slog.Error("player detail query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	if detail == nil {
		response.SetCache(w, config.Env.PlayerCacheTTL)
		response.Error(w, http.StatusNotFound, "not found")
		return
	}

	out := buildPlayerResponse(detail)
	response.SetCache(w, config.Env.PlayerCacheTTL)
	response.JSON(w, http.StatusOK, out)
}

func buildPlayerResponse(d *db.PlayerDetail) playerResponse {
	modes := map[string]*playerModeJSON{
		"author": {GameMode: "author", Scores: []playerScoreJSON{}},
		"gold":   {GameMode: "gold", Scores: []playerScoreJSON{}},
	}

	var totalMedals, totalSkips int64
	var earliest time.Time
	for _, s := range d.Scores {
		createdAt := time.Time{}
		if s.CreatedAt != nil {
			createdAt = *s.CreatedAt
		}
		m, ok := modes[s.GameMode.String()]
		if !ok {
			continue
		}
		m.Scores = append(m.Scores, playerScoreJSON{
			Score:         s.Score,
			MapsCompleted: s.MapsCompleted,
			MapsSkipped:   s.MapsSkipped,
			DurationMs:    s.DurationMs,
			CreatedAt:     createdAt,
		})
		m.RunCount++
		if s.Score > m.BestScore {
			m.BestScore = s.Score
		}
		totalMedals += int64(s.MapsCompleted)
		totalSkips += int64(s.MapsSkipped)
		if !createdAt.IsZero() && (earliest.IsZero() || createdAt.Before(earliest)) {
			earliest = createdAt
		}
	}

	activeSince := ""
	if !earliest.IsZero() {
		activeSince = earliest.Format("2006-01-02")
	}

	return playerResponse{
		Player: playerHeaderJSON{
			OpenplanetID: d.OpenplanetID,
			DisplayName:  d.DisplayName,
		},
		Summary: playerSummaryJSON{
			TotalRuns:    len(d.Scores),
			MedalsEarned: totalMedals,
			MapsSkipped:  totalSkips,
			ActiveSince:  activeSince,
		},
		Modes: []playerModeJSON{*modes["author"], *modes["gold"]},
	}
}
