package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"rmpc-server/api/_pkg/auth"
	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
	"rmpc-server/api/_pkg/validate"
)

type metricRequest struct {
	Name      string `json:"name"      validate:"required"`
	Increment int    `json:"increment"`
}

func Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	auth.RequireAuth(handleMetricSubmit)(w, r)
}

func handleMetricSubmit(w http.ResponseWriter, r *http.Request, _ uuid.UUID) {
	// Parse request
	var req metricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := validate.Struct(req); err != nil {
		response.Error(w, http.StatusBadRequest, validate.FormatError(err))
		return
	}

	if req.Increment <= 0 {
		req.Increment = 1
	}

	// Check allowlist
	if !config.IsAllowedMetric(req.Name) {
		response.Error(w, http.StatusBadRequest, "metric name not allowed")
		return
	}

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	if err := db.UpsertMetric(database, req.Name, req.Increment); err != nil {
		slog.Error("upsert metric error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
