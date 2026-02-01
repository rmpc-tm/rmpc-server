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

func FindSessionByTokenHash(db *sql.DB, tokenHash string) (*model.Sessions, error) {
	stmt := SELECT(
		table.Sessions.AllColumns,
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
	return &dest, nil
}
