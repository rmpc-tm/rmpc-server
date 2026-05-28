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

	// Dedup: ROW_NUMBER() over (player, month) ordered by score gives us each
	// player's best score per month (rn = 1). Using window function here
	// because jet's DISTINCT ON only accepts plain columns, not date_trunc.
	dedupRN := ROW_NUMBER().OVER(
		PARTITION_BY(table.Scores.PlayerID, month).
			ORDER_BY(table.Scores.Score.DESC(), table.Scores.CreatedAt.ASC()),
	)
	dedup := SELECT(
		table.Players.OpenplanetID,
		table.Players.DisplayName,
		month.AS("month_start"),
		table.Scores.Score,
		table.Scores.CreatedAt,
		dedupRN.AS("player_month_rn"),
	).FROM(
		table.Scores.
			INNER_JOIN(table.Players, table.Players.ID.EQ(table.Scores.PlayerID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(
		table.BannedPlayers.ID.IS_NULL().
			AND(table.Scores.GameMode.EQ(modeExpr)).
			AND(table.Scores.CreatedAt.GT_EQ(TimestampzT(earliest))).
			AND(table.Scores.CreatedAt.LT(TimestampzT(before))),
	).AsTable("dedup")

	dOpenplanetID := table.Players.OpenplanetID.From(dedup)
	dDisplayName := table.Players.DisplayName.From(dedup)
	dScore := table.Scores.Score.From(dedup)
	dCreatedAt := table.Scores.CreatedAt.From(dedup)
	dMonthStart := TimestampzColumn("month_start").From(dedup)
	dPlayerMonthRN := IntegerColumn("player_month_rn").From(dedup)

	// Rank players within each month (only the per-player best rows from dedup).
	rankRN := ROW_NUMBER().OVER(
		PARTITION_BY(dMonthStart).
			ORDER_BY(dScore.DESC(), dCreatedAt.ASC()),
	)
	ranked := SELECT(
		dOpenplanetID,
		dDisplayName,
		rankRN.AS("rn"),
	).FROM(dedup).WHERE(
		dPlayerMonthRN.EQ(Int(1)),
	).AsTable("ranked")

	rOpenplanetID := dOpenplanetID.From(ranked)
	rDisplayName := dDisplayName.From(ranked)
	rRN := IntegerColumn("rn").From(ranked)

	// Tally trophies — equivalent to (rn = N)::int + SUM. The explicit cast
	// gives Postgres a typed argument; without it the SUM input is inferred as text.
	gold := SUM(CAST(rRN.EQ(Int(1))).AS_INTEGER())
	silver := SUM(CAST(rRN.EQ(Int(2))).AS_INTEGER())
	bronze := SUM(CAST(rRN.EQ(Int(3))).AS_INTEGER())

	stmt := SELECT(
		rOpenplanetID,
		rDisplayName,
		gold.AS("trophies.gold"),
		silver.AS("trophies.silver"),
		bronze.AS("trophies.bronze"),
	).FROM(ranked).WHERE(
		rRN.LT_EQ(Int(3)),
	).GROUP_BY(
		rOpenplanetID,
		rDisplayName,
	).ORDER_BY(
		gold.DESC(),
		silver.DESC(),
		bronze.DESC(),
		LOWER(rDisplayName).ASC(),
	)

	var entries []HallOfFameRow
	if err := stmt.Query(db, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
