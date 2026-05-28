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

type hofQuery struct {
	GameMode string `json:"game_mode" validate:"required,oneof=author gold"`
}

type hofPlayerJSON struct {
	OpenplanetID string `json:"openplanet_id"`
	DisplayName  string `json:"display_name"`
}

type hofEntryJSON struct {
	Rank   int           `json:"rank"`
	Player hofPlayerJSON `json:"player"`
	Gold   int           `json:"gold"`
	Silver int           `json:"silver"`
	Bronze int           `json:"bronze"`
	Total  int           `json:"total"`
}

type hofResponse struct {
	GameMode string         `json:"game_mode"`
	Entries  []hofEntryJSON `json:"entries"`
}

func HallOfFame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := hofQuery{GameMode: r.URL.Query().Get("game_mode")}
	if err := validate.Struct(query); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	now := time.Now().UTC()
	// TEMP: include the in-progress current month for testing — revert by
	// dropping AddDate(0, 1, 0) so the upper bound is this month's first day.
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// HoF starts one month later than the leaderboard's earliest — the UI archive
	// dropdown hides the pre-launch month, so trophies shouldn't be awarded there.
	hofEarliest := leaderboardEarliestMonth.AddDate(0, 1, 0)
	rows, err := db.GetHallOfFame(database, query.GameMode, hofEarliest, currentMonth)
	if err != nil {
		slog.Error("hall of fame query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Rows arrive pre-sorted by (gold, silver, bronze, name). Assign joint
	// Olympic rank: same (G, S, B) tuple → same rank; otherwise rank = i + 1.
	entries := make([]hofEntryJSON, len(rows))
	rank := 0
	for i, r := range rows {
		if i == 0 || r.Gold != rows[i-1].Gold || r.Silver != rows[i-1].Silver || r.Bronze != rows[i-1].Bronze {
			rank = i + 1
		}
		entries[i] = hofEntryJSON{
			Rank:   rank,
			Player: hofPlayerJSON{OpenplanetID: r.OpenplanetID, DisplayName: r.DisplayName},
			Gold:   r.Gold,
			Silver: r.Silver,
			Bronze: r.Bronze,
			Total:  r.Gold + r.Silver + r.Bronze,
		}
	}

	response.SetCache(w, config.Env.HallOfFameCacheTTL)
	response.JSON(w, http.StatusOK, hofResponse{
		GameMode: query.GameMode,
		Entries:  entries,
	})
}
