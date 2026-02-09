package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
)

type worldRecordJSON struct {
	Score      int32     `json:"score"`
	PlayerName string    `json:"player_name"`
	Date       time.Time `json:"date"`
}

type worldRecordsResponse struct {
	// New structured fields
	AllTime map[string]worldRecordJSON `json:"all_time"`
	Monthly map[string]worldRecordJSON `json:"monthly"`

	// Legacy flat keys for backward compat
	Author *worldRecordJSON `json:"author,omitempty"`
	Gold   *worldRecordJSON `json:"gold,omitempty"`
}

func Worldrecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// All-time world records
	allTime, err := db.GetWorldRecords(database, db.WorldRecordParams{})
	if err != nil {
		slog.Error("world records query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Current month world records
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)
	monthly, err := db.GetWorldRecords(database, db.WorldRecordParams{
		StartTime: &monthStart,
		EndTime:   &monthEnd,
	})
	if err != nil {
		slog.Error("monthly world records query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	allTimeMap := make(map[string]worldRecordJSON, len(allTime))
	for _, r := range allTime {
		createdAt := time.Time{}
		if r.CreatedAt != nil {
			createdAt = *r.CreatedAt
		}
		allTimeMap[r.GameMode] = worldRecordJSON{
			Score:      r.Score,
			PlayerName: r.DisplayName,
			Date:       createdAt,
		}
	}

	monthlyMap := make(map[string]worldRecordJSON, len(monthly))
	for _, r := range monthly {
		createdAt := time.Time{}
		if r.CreatedAt != nil {
			createdAt = *r.CreatedAt
		}
		monthlyMap[r.GameMode] = worldRecordJSON{
			Score:      r.Score,
			PlayerName: r.DisplayName,
			Date:       createdAt,
		}
	}

	out := worldRecordsResponse{
		AllTime: allTimeMap,
		Monthly: monthlyMap,
	}

	// Legacy flat keys
	if v, ok := allTimeMap["author"]; ok {
		out.Author = &v
	}
	if v, ok := allTimeMap["gold"]; ok {
		out.Gold = &v
	}
	if ttl := config.Env.WorldRecordsCacheTTL; ttl > 0 {
		w.Header().Set("Cache-Control",
			fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=60, stale-if-error=3600", int(ttl.Seconds())))
	}

	response.JSON(w, http.StatusOK, out)
}
