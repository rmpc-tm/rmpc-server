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

type hofQuery struct {
	GameMode string `json:"game_mode" validate:"required,oneof=author gold"`
}

type hofPlayerJSON struct {
	OpenplanetID string `json:"openplanet_id"`
	DisplayName  string `json:"display_name"`
	Token        string `json:"t"`
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

// HoF starts one month after the leaderboard's earliest — the UI archive
// dropdown hides the pre-launch month, so trophies aren't awarded there.
var hofEarliestMonth = leaderboardEarliestMonth.AddDate(0, 1, 0)

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
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	rows, err := db.GetHallOfFame(database, query.GameMode, hofEarliestMonth, currentMonth)
	if err != nil {
		slog.Error("hall of fame query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Rows arrive pre-sorted; ranking is fully determined by the query
	// (trophy counts, then best score, then name).
	entries := make([]hofEntryJSON, len(rows))
	for i, r := range rows {
		entries[i] = hofEntryJSON{
			Rank:   i + 1,
			Player: hofPlayerJSON{
				OpenplanetID: r.OpenplanetID,
				DisplayName:  r.DisplayName,
				Token:        playerlink.Sign(r.OpenplanetID),
			},
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
