package config

import (
	"testing"
)

func TestIsAllowedMetric(t *testing.T) {
	// These tests depend on loading config/metrics.yaml relative to this file
	tests := []struct {
		name    string
		metric  string
		allowed bool
	}{
		{"allowed metric", "run_started", true},
		{"allowed metric 2", "map_completed", true},
		{"allowed metric 3", "medal_author", true},
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
