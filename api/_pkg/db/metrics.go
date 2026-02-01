package db

import (
	"database/sql"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/table"
)

func UpsertMetric(db *sql.DB, name string, increment int) error {
	stmt := table.Metrics.INSERT(
		table.Metrics.Name,
		table.Metrics.Count,
	).VALUES(
		name,
		increment,
	).ON_CONFLICT(table.Metrics.Name, table.Metrics.Date).DO_UPDATE(
		SET(
			table.Metrics.Count.SET(
				IntegerExpression(table.Metrics.Count.ADD(Int(int64(increment)))),
			),
		),
	)

	_, err := stmt.Exec(db)
	return err
}
