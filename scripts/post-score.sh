#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

# Dev player tokens
TOKENS=("token-alice" "token-bob" "token-charlie" "token-diana" "token-eve")
GAME_MODES=("author" "gold" "custom")

# Pick a random player
TOKEN=${TOKENS[$((RANDOM % ${#TOKENS[@]}))]}
GAME_MODE=${GAME_MODES[$((RANDOM % ${#GAME_MODES[@]}))]}
SCORE=$((RANDOM % 2200000 + 550000))
MAPS_COMPLETED=$((RANDOM % 20 + 1))
MAPS_SKIPPED=$((RANDOM % 5))
DURATION_MS=$(( (RANDOM % 540 + 60) * 1000 ))  # 60s – 600s

echo "Player: $TOKEN | Mode: $GAME_MODE | Score: $SCORE | Maps: $MAPS_COMPLETED done, $MAPS_SKIPPED skipped | Duration: ${DURATION_MS}ms"

# Authenticate
echo "Authenticating..."
AUTH_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth" \
  -H "Content-Type: application/json" \
  -d "{\"openplanet_token\": \"$TOKEN\"}")

SESSION_TOKEN=$(echo "$AUTH_RESPONSE" | grep -o '"session_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$SESSION_TOKEN" ]; then
  echo "Auth failed: $AUTH_RESPONSE"
  exit 1
fi
echo "Got session token: ${SESSION_TOKEN:0:16}..."

# Submit score
echo "Submitting score..."
curl -s -X POST "$BASE_URL/api/scores" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -d "{
    \"game_mode\": \"$GAME_MODE\",
    \"score\": $SCORE,
    \"maps_completed\": $MAPS_COMPLETED,
    \"maps_skipped\": $MAPS_SKIPPED,
    \"duration_ms\": $DURATION_MS
  }" | python3 -m json.tool 2>/dev/null || echo "(raw response above)"
