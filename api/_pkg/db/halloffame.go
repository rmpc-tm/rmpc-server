package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	. "github.com/go-jet/jet/v2/postgres"

	"rmpc-server/db/.gen/rmpc/public/table"
)

type HallOfFameRow struct {
	OpenplanetID string
	DisplayName  string
	// Months a trophy of each tier was won, oldest first.
	Gold   []time.Time
	Silver []time.Time
	Bronze []time.Time
}

// One month a player finished on the podium.
type hofPodiumRow struct {
	OpenplanetID string    `alias:"players.openplanet_id"`
	DisplayName  string    `alias:"players.display_name"`
	Month        time.Time `alias:"podium.month"`
	Place        int       `alias:"podium.rn"`
	BestScore    int32     `alias:"podium.best_score"`
}

// GetHallOfFame returns players ranked by trophy count for a single game mode
// within [earliest, before). For each month it awards gold/silver/bronze to
// the top 3 best-per-player scores, then aggregates per player. Rows are
// sorted by (gold, silver, bronze, best_score, name) — best_score is the
// player's best within the period, used to break trophy-count ties.
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
		month.AS("month"),
		MAX(table.Scores.Score).AS("best_score"),
		rn.AS("rn"),
	).FROM(
		table.Scores.
			INNER_JOIN(table.Players, table.Players.ID.EQ(table.Scores.PlayerID)).
			LEFT_JOIN(table.BannedPlayers, table.BannedPlayers.PlayerID.EQ(table.Scores.PlayerID)),
	).WHERE(AND(
		table.BannedPlayers.ID.IS_NULL(),
		table.Scores.GameMode.EQ(modeExpr),
		table.Scores.Score.GT(Int(0)),
		table.Scores.CreatedAt.GT_EQ(TimestampzT(earliest)),
		table.Scores.CreatedAt.LT(TimestampzT(before)),
	)).GROUP_BY(
		table.Players.OpenplanetID,
		table.Players.DisplayName,
		month,
	).AsTable("monthly")

	mOpenplanetID := table.Players.OpenplanetID.From(monthly)
	mDisplayName := table.Players.DisplayName.From(monthly)
	mMonth := TimestampzColumn("month").From(monthly)
	mBestScore := IntegerColumn("best_score").From(monthly)
	mRN := IntegerColumn("rn").From(monthly)

	// One row per podium month; the tally per player happens in Go so each
	// trophy keeps the month it was won in.
	stmt := SELECT(
		mOpenplanetID,
		mDisplayName,
		mMonth.AS("podium.month"),
		mRN.AS("podium.rn"),
		mBestScore.AS("podium.best_score"),
	).FROM(monthly).WHERE(
		mRN.LT_EQ(Int(3)),
	).ORDER_BY(
		mMonth.ASC(),
	)

	var podium []hofPodiumRow
	if err := stmt.Query(db, &podium); err != nil {
		return nil, err
	}
	return rankHallOfFame(podium), nil
}

// rankHallOfFame collapses podium months into one row per player, sorted by
// (gold, silver, bronze, best_score, name). Podium rows must be ordered by
// month so each tier's months stay oldest first.
func rankHallOfFame(podium []hofPodiumRow) []HallOfFameRow {
	byPlayer := make(map[string]*HallOfFameRow)
	best := make(map[string]int32)
	order := make([]string, 0, len(podium))

	for _, p := range podium {
		row, ok := byPlayer[p.OpenplanetID]
		if !ok {
			row = &HallOfFameRow{OpenplanetID: p.OpenplanetID, DisplayName: p.DisplayName}
			byPlayer[p.OpenplanetID] = row
			order = append(order, p.OpenplanetID)
		}
		switch p.Place {
		case 1:
			row.Gold = append(row.Gold, p.Month)
		case 2:
			row.Silver = append(row.Silver, p.Month)
		case 3:
			row.Bronze = append(row.Bronze, p.Month)
		}
		if p.BestScore > best[p.OpenplanetID] {
			best[p.OpenplanetID] = p.BestScore
		}
	}

	entries := make([]HallOfFameRow, 0, len(order))
	for _, id := range order {
		entries = append(entries, *byPlayer[id])
	}

	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		for _, c := range [][2]int{
			{len(a.Gold), len(b.Gold)},
			{len(a.Silver), len(b.Silver)},
			{len(a.Bronze), len(b.Bronze)},
		} {
			if c[0] != c[1] {
				return c[0] > c[1]
			}
		}
		if best[a.OpenplanetID] != best[b.OpenplanetID] {
			return best[a.OpenplanetID] > best[b.OpenplanetID]
		}
		return strings.ToLower(a.DisplayName) < strings.ToLower(b.DisplayName)
	})
	return entries
}
