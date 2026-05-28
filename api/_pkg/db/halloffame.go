package db

import (
	"database/sql"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/table"
)

type HallOfFameRow struct {
	OpenplanetID string `alias:"players.openplanet_id"`
	DisplayName  string `alias:"players.display_name"`
	Gold         int    `alias:"trophies.gold"`
	Silver       int    `alias:"trophies.silver"`
	Bronze       int    `alias:"trophies.bronze"`
}

// GetHallOfFame returns players ranked by trophy count for a single game mode
// within [earliest, before). For each month it awards gold/silver/bronze to
// the top 3 best-per-player scores, then aggregates per player. Rows arrive
// pre-sorted by (gold, silver, bronze, name).
//
// Banned players are excluded. gameMode must be "author" or "gold".
func GetHallOfFame(db *sql.DB, gameMode string, earliest, before time.Time) ([]HallOfFameRow, error) {
	modeExpr, ok := gameModeExpression[gameMode]
	if !ok {
		return nil, fmt.Errorf("invalid game mode: %s", gameMode)
	}

	month := DATE_TRUNC(MONTH, table.Scores.CreatedAt, "UTC")

	// One pass: GROUP BY collapses each player's monthly scores into a single
	// row (their best for that month); the window function then ranks players
	// within the month using those aggregates. Window functions run after
	// GROUP BY, so MAX/MIN are valid inside ORDER BY.
	rn := ROW_NUMBER().OVER(
		PARTITION_BY(month).
			ORDER_BY(MAX(table.Scores.Score).DESC(), MIN(table.Scores.CreatedAt).ASC()),
	)
	monthly := SELECT(
		table.Players.OpenplanetID,
		table.Players.DisplayName,
		rn.AS("rn"),
	).FROM(
		table.Scores.
			INNER_JOIN(table.Players, table.Players.ID.EQ(table.Scores.PlayerID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(
		table.BannedPlayers.ID.IS_NULL().
			AND(table.Scores.GameMode.EQ(modeExpr)).
			AND(table.Scores.CreatedAt.GT_EQ(TimestampzT(earliest))).
			AND(table.Scores.CreatedAt.LT(TimestampzT(before))),
	).GROUP_BY(
		table.Players.OpenplanetID,
		table.Players.DisplayName,
		month,
	).AsTable("monthly")

	mOpenplanetID := table.Players.OpenplanetID.From(monthly)
	mDisplayName := table.Players.DisplayName.From(monthly)
	mRN := IntegerColumn("rn").From(monthly)

	// Tally trophies. COUNT ignores NULLs, so the CASE returns 1 for matches
	// and NULL (no ELSE) otherwise — equivalent to COUNT(*) FILTER (WHERE rn = N),
	// which jet doesn't expose.
	gold := COUNT(CASE().WHEN(mRN.EQ(Int(1))).THEN(Int(1)))
	silver := COUNT(CASE().WHEN(mRN.EQ(Int(2))).THEN(Int(1)))
	bronze := COUNT(CASE().WHEN(mRN.EQ(Int(3))).THEN(Int(1)))

	stmt := SELECT(
		mOpenplanetID,
		mDisplayName,
		gold.AS("trophies.gold"),
		silver.AS("trophies.silver"),
		bronze.AS("trophies.bronze"),
	).FROM(monthly).WHERE(
		mRN.LT_EQ(Int(3)),
	).GROUP_BY(
		mOpenplanetID,
		mDisplayName,
	).ORDER_BY(
		gold.DESC(),
		silver.DESC(),
		bronze.DESC(),
		LOWER(mDisplayName).ASC(),
	)

	var entries []HallOfFameRow
	if err := stmt.Query(db, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
