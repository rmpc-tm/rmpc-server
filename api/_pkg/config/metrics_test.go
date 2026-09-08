package config

import (
	"testing"
)

func TestIsAllowedMetric(t *testing.T) {
	// The allow-list is embedded from metrics.yaml in this package. The
	// medal_* names are not on it.
	tests := []struct {
		name    string
		metric  string
		allowed bool
	}{
		{"allowed metric", "run_started", true},
		{"allowed metric 2", "map_completed", true},
		{"allowed metric 3", "map_broken", true},
		{"medal metric not on the list", "medal_author", false},
		{"disallowed metric", "nonexistent_metric", false},
		{"empty string", "", false},
		{"sql injection attempt", "'; DROP TABLE metrics; --", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowedMetric(tt.metric)
			if got != tt.allowed {
				t.Errorf("IsAllowedMetric(%q) = %v, want %v", tt.metric, got, tt.allowed)
			}
		})
	}
}

// An empty allow-list rejects every metric without erroring anywhere, so assert
// the list actually loaded.
func TestAllowListIsPopulated(t *testing.T) {
	metricsOnce.Do(loadMetrics)

	if len(allowedMetrics) == 0 {
		t.Fatal("allow-list is empty: metrics.yaml did not load, every metric would be rejected")
	}
}
