package handler

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"rmpc-server/api/_pkg/config"
	"rmpc-server/api/_pkg/db"
	"rmpc-server/api/_pkg/response"
)

type hofPlayerJSON struct {
	OpenplanetID string `json:"openplanet_id"`
	DisplayName  string `json:"display_name"`
}

type hofEntryJSON struct {
	Rank   int           `json:"rank"`
	Player hofPlayerJSON `json:"player"`
	Gold   int           `json:"gold"`
	Silver int           `json:"silver"`
	Bronze int           `json:"bronze"`
	Total  int           `json:"total"`
}

type hofModeJSON struct {
	GameMode string         `json:"game_mode"`
	Entries  []hofEntryJSON `json:"entries"`
}

type hofResponse struct {
	Modes         []hofModeJSON `json:"modes"`
	MonthsCounted int           `json:"months_counted"`
	EarliestMonth string        `json:"earliest_month"`
	LatestMonth   string        `json:"latest_month,omitempty"`
}

func HallOfFame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	earliest := leaderboardEarliestMonth

	database, err := db.GetDB()
	if err != nil {
		slog.Error("database connection error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	rows, err := db.GetMonthlyScores(database, earliest, currentMonth)
	if err != nil {
		slog.Error("hall of fame query error", "error", err)
		response.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	modeOrder := []string{"author", "gold"}

	// Reduce to best score per (mode, month, player), then group by (mode, month).
	type bestKey struct {
		Mode       string
		MonthStart time.Time
		Openplanet string
	}
	type bestVal struct {
		Score       int32
		DisplayName string
	}
	best := map[bestKey]bestVal{}
	for _, r := range rows {
		mode := r.GameMode.String()
		if _, ok := map[string]struct{}{"author": {}, "gold": {}}[mode]; !ok {
			continue
		}
		monthStart := time.Date(r.CreatedAt.UTC().Year(), r.CreatedAt.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
		k := bestKey{Mode: mode, MonthStart: monthStart, Openplanet: r.OpenplanetID}
		if existing, ok := best[k]; !ok || r.Score > existing.Score {
			best[k] = bestVal{Score: r.Score, DisplayName: r.DisplayName}
		}
	}

	type playerScore struct {
		OpenplanetID string
		DisplayName  string
		Score        int32
	}
	groups := map[string]map[time.Time][]playerScore{
		"author": {},
		"gold":   {},
	}
	for k, v := range best {
		groups[k.Mode][k.MonthStart] = append(groups[k.Mode][k.MonthStart], playerScore{
			OpenplanetID: k.Openplanet,
			DisplayName:  v.DisplayName,
			Score:        v.Score,
		})
	}

	// Award trophies: top 3 per (mode, month) by score desc.
	type tally struct {
		Name   string
		Gold   int
		Silver int
		Bronze int
	}
	buckets := map[string]map[string]*tally{
		"author": {},
		"gold":   {},
	}
	monthSeen := map[string]bool{}

	for _, mode := range modeOrder {
		for monthStart, players := range groups[mode] {
			sort.Slice(players, func(i, j int) bool {
				return players[i].Score > players[j].Score
			})
			if len(players) > 0 {
				monthSeen[monthStart.Format("2006-01")] = true
			}
			for i, p := range players {
				if i >= 3 {
					break
				}
				t := buckets[mode][p.OpenplanetID]
				if t == nil {
					t = &tally{}
					buckets[mode][p.OpenplanetID] = t
				}
				t.Name = p.DisplayName
				switch i {
				case 0:
					t.Gold++
				case 1:
					t.Silver++
				case 2:
					t.Bronze++
				}
			}
		}
	}

	modes := make([]hofModeJSON, 0, len(modeOrder))
	for _, mode := range modeOrder {
		type rankRow struct {
			OpenplanetID string
			Name         string
			G, S, B      int
		}
		flat := make([]rankRow, 0, len(buckets[mode]))
		for opid, t := range buckets[mode] {
			flat = append(flat, rankRow{OpenplanetID: opid, Name: t.Name, G: t.Gold, S: t.Silver, B: t.Bronze})
		}
		sort.Slice(flat, func(i, j int) bool {
			a, b := flat[i], flat[j]
			if a.G != b.G {
				return a.G > b.G
			}
			if a.S != b.S {
				return a.S > b.S
			}
			if a.B != b.B {
				return a.B > b.B
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		})

		entries := make([]hofEntryJSON, len(flat))
		rank := 0
		for i, row := range flat {
			if i == 0 || row.G != flat[i-1].G || row.S != flat[i-1].S || row.B != flat[i-1].B {
				rank = i + 1
			}
			entries[i] = hofEntryJSON{
				Rank: rank,
				Player: hofPlayerJSON{
					OpenplanetID: row.OpenplanetID,
					DisplayName:  row.Name,
				},
				Gold:   row.G,
				Silver: row.S,
				Bronze: row.B,
				Total:  row.G + row.S + row.B,
			}
		}

		modes = append(modes, hofModeJSON{
			GameMode: mode,
			Entries:  entries,
		})
	}

	out := hofResponse{
		Modes:         modes,
		MonthsCounted: len(monthSeen),
		EarliestMonth: earliest.Format("2006-01"),
	}
	if lastCompleted := currentMonth.AddDate(0, -1, 0); !lastCompleted.Before(earliest) {
		out.LatestMonth = lastCompleted.Format("2006-01")
	}

	response.SetCache(w, config.Env.HallOfFameCacheTTL)
	response.JSON(w, http.StatusOK, out)
}
