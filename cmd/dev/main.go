package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	handler "rmpc-server/api"
)

const DevOpenplanetToken = "dev-token"

func main() {
	// Start mock Openplanet auth server
	mockAddr := os.Getenv("MOCK_OPENPLANET_ADDR")
	if mockAddr == "" {
		mockAddr = ":8081"
	}
	go startMockOpenplanet(mockAddr)

	// Start main server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth", handler.Auth)
	mux.HandleFunc("/api/scores", handler.Scores)
	mux.HandleFunc("/api/leaderboard", handler.Leaderboard)
	mux.HandleFunc("/api/metrics", handler.Metrics)
	mux.Handle("/", http.FileServer(http.Dir("public")))

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	slog.Info("listening", "addr", addr)
	slog.Info("mock openplanet", "addr", mockAddr, "valid_token", DevOpenplanetToken)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func startMockOpenplanet(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/validate", handleMockValidate)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("mock openplanet error", "error", err)
		os.Exit(1)
	}
}

func handleMockValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token  string `json:"token"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Token != DevOpenplanetToken {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"account_id":   "dev-player-001",
		"display_name": "DevPlayer",
	})
}
