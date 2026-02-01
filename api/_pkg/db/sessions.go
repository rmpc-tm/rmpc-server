package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

	"rmpc-server/db/.gen/rmpc/public/model"
	"rmpc-server/db/.gen/rmpc/public/table"
)

func CreateSession(db *sql.DB, playerID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	// Delete existing sessions for this player (one session per player)
	delStmt := table.Sessions.DELETE().WHERE(
		table.Sessions.PlayerID.EQ(UUID(playerID)),
	)
	if _, err := delStmt.Exec(db); err != nil {
		return err
	}

	stmt := table.Sessions.INSERT(
		table.Sessions.PlayerID,
		table.Sessions.TokenHash,
		table.Sessions.ExpiresAt,
	).VALUES(
		playerID,
		tokenHash,
		expiresAt,
	)

	_, err := stmt.Exec(db)
	return err
}

type Session struct {
	ID        uuid.UUID
	PlayerID  uuid.UUID
	ExpiresAt time.Time
}

func FindSessionByTokenHash(db *sql.DB, tokenHash string) (*Session, error) {
	stmt := SELECT(
		table.Sessions.ID,
		table.Sessions.PlayerID,
		table.Sessions.ExpiresAt,
	).FROM(
		table.Sessions,
	).WHERE(
		table.Sessions.TokenHash.EQ(String(tokenHash)),
	).LIMIT(1)

	var dest model.Sessions
	err := stmt.Query(db, &dest)
	if err != nil {
		if err.Error() == "jet: sql: no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &Session{
		ID:        dest.ID,
		PlayerID:  dest.PlayerID,
		ExpiresAt: dest.ExpiresAt,
	}, nil
}
