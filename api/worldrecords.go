package handler

import (
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
)

type worldRecordJSON struct {
	Score      int32     `json:"score"`
	PlayerName string    `json:"player_name"`
	Date       time.Time `json:"date"`
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

	records, err := db.GetWorldRecords(database)
	if err != nil {
		slog.Error("world records query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	out := make(map[string]worldRecordJSON, len(records))
	for _, r := range records {
		createdAt := time.Time{}
		if r.CreatedAt != nil {
			createdAt = *r.CreatedAt
		}
		out[r.GameMode] = worldRecordJSON{
			Score:      r.Score,
			PlayerName: r.DisplayName,
			Date:       createdAt,
		}
	}

	response.JSON(w, http.StatusOK, out)
}
