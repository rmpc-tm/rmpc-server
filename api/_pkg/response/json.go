package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func SetCache(w http.ResponseWriter, ttl time.Duration) {
	if ttl > 0 {
		w.Header().Set("Cache-Control",
			fmt.Sprintf("public, s-maxage=%d, stale-while-revalidate=60, stale-if-error=3600", int(ttl.Seconds())))
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}
