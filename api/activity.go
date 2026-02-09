package handler

import (
	"log/slog"
	"net/http"
	"time"

	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
)

type activityResponse struct {
	Medals []int64 `json:"medals"`
}

func Activity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	const days = 30

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	lookup, err := db.GetMedalActivity(database, days)
	if err != nil {
		slog.Error("activity query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	// Fill in all days in the range
	start := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)

	medals := make([]int64, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		medals[i] = lookup[d.Format("2006-01-02")]
	}

	response.SetCache(w, config.Env.ActivityCacheTTL)

	response.JSON(w, http.StatusOK, activityResponse{
		Medals: medals,
	})
}
