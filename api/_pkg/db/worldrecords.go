package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/enum"
	"rmpc-server/db/.gen/rmpc/public/table"
)

type WorldRecord struct {
	GameMode    string     `alias:"scores.game_mode"`
	Score       int32      `alias:"scores.score"`
	DisplayName string     `alias:"players.display_name"`
	OpenplanetID string    `alias:"players.openplanet_id"`
	CreatedAt   *time.Time `alias:"scores.created_at"`
}

func GetWorldRecords(db *sql.DB) ([]WorldRecord, error) {
	// Best score per game_mode using DISTINCT ON, excluding banned players
	stmt := SELECT(
		table.Scores.GameMode,
		table.Scores.Score,
		table.Players.DisplayName,
		table.Players.OpenplanetID,
		table.Scores.CreatedAt,
	).DISTINCT(
		table.Scores.GameMode,
	).FROM(
		table.Scores.
			INNER_JOIN(table.Players, table.Players.ID.EQ(table.Scores.PlayerID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(
		table.BannedPlayers.ID.IS_NULL().AND(
			table.Scores.GameMode.IN(enum.GameMode.Author, enum.GameMode.Gold),
		),
	).ORDER_BY(
		table.Scores.GameMode,
		table.Scores.Score.DESC(),
	)

	var records []WorldRecord
	err := stmt.Query(db, &records)
	if err != nil {
		return nil, err
	}

	return records, nil
}
