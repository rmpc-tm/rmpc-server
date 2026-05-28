package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"rmpc-server/db/.gen/rmpc/public/enum"
	"rmpc-server/db/.gen/rmpc/public/model"
	"rmpc-server/db/.gen/rmpc/public/table"
)

type MonthlyScoreRow struct {
	PlayerID     uuid.UUID      `alias:"scores.player_id"`
	GameMode     model.GameMode `alias:"scores.game_mode"`
	Score        int32          `alias:"scores.score"`
	CreatedAt    time.Time      `alias:"scores.created_at"`
	OpenplanetID string         `alias:"players.openplanet_id"`
	DisplayName  string         `alias:"players.display_name"`
}

// GetMonthlyScores returns all (non-banned) scores in the author and gold game
// modes within [earliest, before). The caller is expected to bucket by month
// and pick the per-player best in code — the trophy aggregation logic lives in
// the handler.
func GetMonthlyScores(db *sql.DB, earliest, before time.Time) ([]MonthlyScoreRow, error) {
	stmt := SELECT(
		table.Scores.PlayerID,
		table.Scores.GameMode,
		table.Scores.Score,
		table.Scores.CreatedAt,
		table.Players.OpenplanetID,
		table.Players.DisplayName,
	).FROM(
		table.Scores.
			INNER_JOIN(table.Players, table.Players.ID.EQ(table.Scores.PlayerID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(
		table.BannedPlayers.ID.IS_NULL().
			AND(table.Scores.GameMode.IN(enum.GameMode.Author, enum.GameMode.Gold)).
			AND(table.Scores.CreatedAt.GT_EQ(TimestampzT(earliest))).
			AND(table.Scores.CreatedAt.LT(TimestampzT(before))),
	)

	var rows []MonthlyScoreRow
	if err := stmt.Query(db, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
