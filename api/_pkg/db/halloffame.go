package db

import (
	"database/sql"
	"time"
)

type HallOfFameRow struct {
	OpenplanetID string
	DisplayName  string
	Gold         int
	Silver       int
	Bronze       int
}

// GetHallOfFame returns players ranked by trophy count for a single game mode
// within [earliest, before). For each completed month in the range it awards
// gold/silver/bronze to the top 3 best-per-player scores, then aggregates per
// player. Rows come back already sorted by (gold, silver, bronze, name).
//
// Banned players are excluded. gameMode must be a valid scores.game_mode value
// (caller validates).
func GetHallOfFame(db *sql.DB, gameMode string, earliest, before time.Time) ([]HallOfFameRow, error) {
	const q = `
WITH best AS (
    SELECT DISTINCT ON (s.player_id, date_trunc('month', s.created_at AT TIME ZONE 'UTC'))
        p.openplanet_id,
        p.display_name,
        date_trunc('month', s.created_at AT TIME ZONE 'UTC') AS month_start,
        s.score,
        s.created_at
    FROM scores s
    INNER JOIN players p ON p.id = s.player_id
    LEFT  JOIN banned_players b ON b.player_id = s.player_id
    WHERE b.id IS NULL
      AND s.game_mode = $1::game_mode
      AND s.created_at >= $2
      AND s.created_at <  $3
    ORDER BY s.player_id,
             date_trunc('month', s.created_at AT TIME ZONE 'UTC'),
             s.score DESC,
             s.created_at ASC
),
ranked AS (
    SELECT openplanet_id, display_name,
           ROW_NUMBER() OVER (
               PARTITION BY month_start
               ORDER BY score DESC, created_at ASC
           ) AS rn
    FROM best
)
SELECT openplanet_id, display_name,
       SUM((rn = 1)::int) AS gold,
       SUM((rn = 2)::int) AS silver,
       SUM((rn = 3)::int) AS bronze
FROM ranked
WHERE rn <= 3
GROUP BY openplanet_id, display_name
ORDER BY gold DESC, silver DESC, bronze DESC, lower(display_name) ASC;
`

	rows, err := db.Query(q, gameMode, earliest, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HallOfFameRow
	for rows.Next() {
		var r HallOfFameRow
		if err := rows.Scan(&r.OpenplanetID, &r.DisplayName, &r.Gold, &r.Silver, &r.Bronze); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
