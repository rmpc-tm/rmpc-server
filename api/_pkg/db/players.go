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

func UpsertPlayer(db *sql.DB, openplanetID, displayName string) (uuid.UUID, error) {
	stmt := table.Players.INSERT(
		table.Players.OpenplanetID,
		table.Players.DisplayName,
	).VALUES(
		openplanetID,
		displayName,
	).ON_CONFLICT(table.Players.OpenplanetID).DO_UPDATE(
		SET(
			table.Players.DisplayName.SET(String(displayName)),
			table.Players.UpdatedAt.SET(TimestampzExpression(NOW())),
		),
	).RETURNING(table.Players.ID)

	var dest model.Players
	err := stmt.Query(db, &dest)
	return dest.ID, err
}

type PlayerScoreRow struct {
	GameMode      model.GameMode `alias:"scores.game_mode"`
	Score         int32          `alias:"scores.score"`
	MapsCompleted int32          `alias:"scores.maps_completed"`
	MapsSkipped   int32          `alias:"scores.maps_skipped"`
	DurationMs    int32          `alias:"scores.duration_ms"`
	CreatedAt     *time.Time     `alias:"scores.created_at"`
}

type PlayerDetail struct {
	OpenplanetID string
	DisplayName  string
	Scores       []PlayerScoreRow
}

// GetPlayerDetail returns a player and all their author/gold scores ordered
// newest first. Returns (nil, nil) when the player doesn't exist, is banned,
// or has no scores in these modes.
func GetPlayerDetail(db *sql.DB, openplanetID string) (*PlayerDetail, error) {
	stmt := SELECT(
		table.Players.OpenplanetID,
		table.Players.DisplayName,
		table.Scores.GameMode,
		table.Scores.Score,
		table.Scores.MapsCompleted,
		table.Scores.MapsSkipped,
		table.Scores.DurationMs,
		table.Scores.CreatedAt,
	).FROM(
		table.Players.
			INNER_JOIN(table.Scores, table.Scores.PlayerID.EQ(table.Players.ID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Players.ID)),
	).WHERE(AND(
		table.Players.OpenplanetID.EQ(String(openplanetID)),
		table.BannedPlayers.ID.IS_NULL(),
		table.Scores.GameMode.IN(enum.GameMode.Author, enum.GameMode.Gold),
	)).ORDER_BY(
		table.Scores.CreatedAt.DESC(),
	)

	var rows []struct {
		OpenplanetID string `alias:"players.openplanet_id"`
		DisplayName  string `alias:"players.display_name"`
		PlayerScoreRow
	}
	if err := stmt.Query(db, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	detail := &PlayerDetail{
		OpenplanetID: rows[0].OpenplanetID,
		DisplayName:  rows[0].DisplayName,
		Scores:       make([]PlayerScoreRow, len(rows)),
	}
	for i, r := range rows {
		detail.Scores[i] = r.PlayerScoreRow
	}
	return detail, nil
}
