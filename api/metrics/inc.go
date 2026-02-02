package handler

import (
	"net/http"

	api "rmpc-server/api"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	api.MetricInc(w, r, r.URL.Query().Get("name"))
}
