package config

import (
	"os"
	"time"
)

// Env holds all environment-driven configuration, loaded once at init.
// Defaults are applied when the env var is missing or invalid.
var Env struct {
	// DATABASE_URL - PostgreSQL connection string (required)
	DatabaseURL string

	// OPENPLANET_PLUGIN_SECRET - secret for Openplanet token validation (required)
	OpenplanetPluginSecret string

	// OPENPLANET_AUTH_URL - Openplanet auth endpoint (default: https://openplanet.dev/api/auth/validate)
	OpenplanetAuthURL string

	// SESSION_TOKEN_EXPIRY - session lifetime as Go duration, e.g. "720h" (default: 720h)
	SessionTokenExpiry time.Duration

	// SCORE_COOLDOWN - minimum time between score submissions per player, e.g. "1m" (default: 1m)
	ScoreCooldown time.Duration

	// AUTH_RATE_LIMIT - max auth requests per IP per minute (default: 10)
	AuthRateLimit int
}

func init() {
	Env.DatabaseURL = os.Getenv("DATABASE_URL")
	Env.OpenplanetPluginSecret = os.Getenv("OPENPLANET_PLUGIN_SECRET")
	Env.OpenplanetAuthURL = stringEnv("OPENPLANET_AUTH_URL", "https://openplanet.dev/api/auth/validate")
	Env.SessionTokenExpiry = durationEnv("SESSION_TOKEN_EXPIRY", 24*time.Hour)
	Env.ScoreCooldown = durationEnv("SCORE_COOLDOWN", 15*time.Minute)
	Env.AuthRateLimit = 10
}

func stringEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

