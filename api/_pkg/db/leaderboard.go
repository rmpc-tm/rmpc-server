package db

import (
	"database/sql"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"rmpc-server/db/.gen/rmpc/public/enum"
	"rmpc-server/db/.gen/rmpc/public/model"
	"rmpc-server/db/.gen/rmpc/public/table"
)

var gameModeExpression = map[string]StringExpression{
	"author": enum.GameMode.Author,
	"gold":   enum.GameMode.Gold,
}

type LeaderboardEntry struct {
	Rank          int            `sql:"-"`
	PlayerID      uuid.UUID      `alias:"scores.player_id"`
	OpenplanetID  string         `alias:"players.openplanet_id"`
	DisplayName   string         `alias:"players.display_name"`
	Score         int32          `alias:"scores.score"`
	MapsCompleted int32          `alias:"scores.maps_completed"`
	MapsSkipped   int32          `alias:"scores.maps_skipped"`
	DurationMs    int32          `alias:"scores.duration_ms"`
	GameMode      model.GameMode `alias:"scores.game_mode"`
	CreatedAt     *time.Time     `alias:"scores.created_at"`
}

type LeaderboardParams struct {
	GameMode  string
	StartTime *time.Time
	EndTime   *time.Time
}

func GetLeaderboard(db *sql.DB, params LeaderboardParams) ([]LeaderboardEntry, error) {
	condition := table.BannedPlayers.ID.IS_NULL()

	if params.GameMode != "" {
		expr, ok := gameModeExpression[params.GameMode]
		if !ok {
			return nil, fmt.Errorf("invalid game mode: %s", params.GameMode)
		}
		condition = condition.AND(table.Scores.GameMode.EQ(expr))
	}
	if params.StartTime != nil {
		condition = condition.AND(table.Scores.CreatedAt.GT_EQ(TimestampzT(*params.StartTime)))
	}
	if params.EndTime != nil {
		condition = condition.AND(table.Scores.CreatedAt.LT(TimestampzT(*params.EndTime)))
	}

	// Best score per player using DISTINCT ON
	bestScores := SELECT(
		table.Scores.PlayerID,
		table.Scores.Score,
		table.Scores.MapsCompleted,
		table.Scores.MapsSkipped,
		table.Scores.DurationMs,
		table.Scores.GameMode,
		table.Scores.CreatedAt,
		table.Players.OpenplanetID,
		table.Players.DisplayName,
	).DISTINCT(
		table.Scores.PlayerID,
	).FROM(
		table.Scores.
			INNER_JOIN(table.Players, table.Players.ID.EQ(table.Scores.PlayerID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(
		condition,
	).ORDER_BY(
		table.Scores.PlayerID,
		table.Scores.Score.DESC(),
	).AsTable("best_scores")

	// Columns from the CTE
	bsPlayerID := table.Scores.PlayerID.From(bestScores)
	bsOpenplanetID := table.Players.OpenplanetID.From(bestScores)
	bsDisplayName := table.Players.DisplayName.From(bestScores)
	bsScore := table.Scores.Score.From(bestScores)
	bsMapsCompleted := table.Scores.MapsCompleted.From(bestScores)
	bsMapsSkipped := table.Scores.MapsSkipped.From(bestScores)
	bsDurationMs := table.Scores.DurationMs.From(bestScores)
	bsGameMode := table.Scores.GameMode.From(bestScores)
	bsCreatedAt := table.Scores.CreatedAt.From(bestScores)

	stmt := SELECT(
		bsPlayerID,
		bsOpenplanetID,
		bsDisplayName,
		bsScore,
		bsMapsCompleted,
		bsMapsSkipped,
		bsDurationMs,
		bsGameMode,
		bsCreatedAt,
	).FROM(
		bestScores,
	).ORDER_BY(
		bsScore.DESC(),
	).LIMIT(100)

	var entries []LeaderboardEntry
	err := stmt.Query(db, &entries)
	if err != nil {
		return nil, err
	}

	for i := range entries {
		entries[i].Rank = i + 1
	}

	return entries, nil
}
