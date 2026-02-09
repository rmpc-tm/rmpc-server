package db

import (
	"database/sql"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/table"
)

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

	query, args := stmt.Sql()
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var date time.Time
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		result[date.Format("2006-01-02")] = count
	}
	return result, rows.Err()
}
