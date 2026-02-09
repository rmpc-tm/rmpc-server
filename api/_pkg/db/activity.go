package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/table"
)

type activityRow struct {
	Date  time.Time `alias:"bucket"`
	Count int64     `alias:"total_medals"`
}

// GetMedalActivity returns the total maps completed per day for the last N days,
// keyed by date string (YYYY-MM-DD). Days with no activity are absent from the map.
func GetMedalActivity(db *sql.DB, days int) (map[string]int64, error) {
	bucket := CAST(table.Scores.CreatedAt).AS_DATE()

	stmt := SELECT(
		bucket.AS("bucket"),
		SUM(table.Scores.MapsCompleted).AS("total_medals"),
	).FROM(
		table.Scores,
	).WHERE(
		table.Scores.CreatedAt.GT_EQ(NOW().SUB(INTERVAL(float64(days), DAY))),
	).GROUP_BY(
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
		result[r.Date.Format("2006-01-02")] = r.Count
	}
	return result, nil
}
