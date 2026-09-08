package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/table"
)

type activityRow struct {
	Date  time.Time `alias:"medals.bucket"`
	Count int64     `alias:"medals.total"`
}

// GetMedalActivity returns total maps completed per UTC day from `since` onwards,
// keyed as YYYY-MM-DD; days with no activity are absent. `since` should be a UTC
// midnight. Grouping and key formatting are both pinned to UTC.
func GetMedalActivity(db *sql.DB, since time.Time) (map[string]int64, error) {
	bucket := DATE_TRUNC(DAY, table.Scores.CreatedAt, "UTC")

	stmt := SELECT(
		bucket.AS("medals.bucket"),
		SUM(table.Scores.MapsCompleted).AS("medals.total"),
	).FROM(
		table.Scores.
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(AND(
		table.BannedPlayers.ID.IS_NULL(),
		table.Scores.CreatedAt.GT_EQ(TimestampzT(since)),
	)).GROUP_BY(
		bucket,
	).ORDER_BY(
		bucket.ASC(),
	)

	var rows []activityRow
	if err := stmt.Query(db, &rows); err != nil {
		return nil, err
	}

	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Date.UTC().Format("2006-01-02")] = r.Count
	}
	return result, nil
}
