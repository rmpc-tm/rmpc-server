package db

import (
	"database/sql"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"

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

	var dest struct {
		ID uuid.UUID
	}
	err := stmt.Query(db, &dest)
	return dest.ID, err
}
