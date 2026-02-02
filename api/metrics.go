package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"rmpc-server/api/_pkg/auth"
	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
)

// MetricInc handles POST /api/metrics/inc?name=X
func MetricInc(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	auth.RequireAuth(func(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
		if !config.IsAllowedMetric(name) {
			response.Error(w, http.StatusBadRequest, "metric name not allowed")
			return
		}

		database, err := db.GetDB()
		if err != nil {
			slog.Error("database connection error", "error", err)
			response.Error(w, http.StatusServiceUnavailable, "service unavailable")
			return
		}

		if err := db.UpsertMetric(database, name, 1); err != nil {
			slog.Error("upsert metric error", "error", err)
			response.Error(w, http.StatusServiceUnavailable, "service unavailable")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})(w, r)
}
