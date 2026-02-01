package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"rmpc-server/db/.gen/rmpc/public/model"
	"rmpc-server/db/.gen/rmpc/public/table"
)

type ScoreInput struct {
	PlayerID      uuid.UUID
	GameMode      string
	Score         int32
	MapsCompleted int32
	MapsSkipped   int32
	DurationMs    int32
	Metadata      *string
}

func InsertScore(db *sql.DB, input ScoreInput) (uuid.UUID, time.Time, error) {
	stmt := table.Scores.INSERT(
		table.Scores.PlayerID,
		table.Scores.GameMode,
		table.Scores.Score,
		table.Scores.MapsCompleted,
		table.Scores.MapsSkipped,
		table.Scores.DurationMs,
		table.Scores.Metadata,
	).VALUES(
		input.PlayerID,
		input.GameMode,
		input.Score,
		input.MapsCompleted,
		input.MapsSkipped,
		input.DurationMs,
		input.Metadata,
	).RETURNING(
		table.Scores.ID,
		table.Scores.CreatedAt,
	)

	var dest model.Scores
	err := stmt.Query(db, &dest)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}

	createdAt := time.Time{}
	if dest.CreatedAt != nil {
		createdAt = *dest.CreatedAt
	}
	return dest.ID, createdAt, nil
}

func CanSubmitScore(db *sql.DB, playerID uuid.UUID, cooldown time.Duration) (bool, error) {
	stmt := SELECT(
		COUNT(STAR),
	).FROM(
		table.Scores,
	).WHERE(
		table.Scores.PlayerID.EQ(UUID(playerID)).
			AND(table.Scores.CreatedAt.GT(
				TimestampzExpression(NOW().SUB(INTERVALd(cooldown))),
			)),
	)

	var dest struct {
		Count int64
	}
	err := stmt.Query(db, &dest)
	if err != nil {
		return false, err
	}
	return dest.Count == 0, nil
}

func IsPlayerBanned(db *sql.DB, playerID uuid.UUID) (bool, error) {
	stmt := SELECT(
		COUNT(STAR),
	).FROM(
		table.BannedPlayers,
	).WHERE(
		table.BannedPlayers.PlayerID.EQ(UUID(playerID)),
	)

	var dest struct {
		Count int64
	}
	err := stmt.Query(db, &dest)
	if err != nil {
		return false, err
	}
	return dest.Count > 0, nil
}
