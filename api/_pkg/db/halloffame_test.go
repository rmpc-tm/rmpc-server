package db

import (
	"testing"
	"time"
)

func month(y int, m time.Month) time.Time {
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}

func podium(id, name string, place int, best int32, m time.Time) hofPodiumRow {
	return hofPodiumRow{
		OpenplanetID: id,
		DisplayName:  name,
		Month:        m,
		Place:        place,
		BestScore:    best,
	}
}

func TestRankHallOfFame(t *testing.T) {
	jan, feb := month(2026, time.January), month(2026, time.February)

	tests := []struct {
		name  string
		rows  []hofPodiumRow
		order []string
	}{
		{
			name: "more gold wins over more trophies overall",
			rows: []hofPodiumRow{
				podium("a", "Alice", 1, 900, jan),
				podium("b", "Bob", 2, 800, jan),
				podium("b", "Bob", 2, 700, feb),
			},
			order: []string{"a", "b"},
		},
		{
			name: "equal gold falls through to silver then bronze",
			rows: []hofPodiumRow{
				podium("a", "Alice", 1, 900, jan),
				podium("a", "Alice", 3, 500, feb),
				podium("b", "Bob", 1, 800, feb),
				podium("b", "Bob", 2, 600, jan),
			},
			order: []string{"b", "a"},
		},
		{
			name: "equal trophies broken by best score, then by name",
			rows: []hofPodiumRow{
				podium("a", "Alice", 1, 700, jan),
				podium("b", "Bob", 1, 900, feb),
				podium("c", "cara", 1, 700, feb),
			},
			order: []string{"b", "a", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := rankHallOfFame(tt.rows)
			if len(entries) != len(tt.order) {
				t.Fatalf("got %d entries, want %d", len(entries), len(tt.order))
			}
			for i, want := range tt.order {
				if entries[i].OpenplanetID != want {
					t.Errorf("position %d: got %q, want %q", i, entries[i].OpenplanetID, want)
				}
			}
		})
	}
}

func TestRankHallOfFameCollectsMonthsPerTier(t *testing.T) {
	jan, feb, mar := month(2026, time.January), month(2026, time.February), month(2026, time.March)

	entries := rankHallOfFame([]hofPodiumRow{
		podium("a", "Alice", 1, 900, jan),
		podium("a", "Alice", 3, 500, feb),
		podium("a", "Alice", 1, 800, mar),
	})

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	got := entries[0]
	if len(got.Gold) != 2 || !got.Gold[0].Equal(jan) || !got.Gold[1].Equal(mar) {
		t.Errorf("gold months: got %v, want [%v %v]", got.Gold, jan, mar)
	}
	if len(got.Silver) != 0 {
		t.Errorf("silver months: got %v, want none", got.Silver)
	}
	if len(got.Bronze) != 1 || !got.Bronze[0].Equal(feb) {
		t.Errorf("bronze months: got %v, want [%v]", got.Bronze, feb)
	}
}
