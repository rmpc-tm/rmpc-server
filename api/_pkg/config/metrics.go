package config

import (
	_ "embed"
	"log/slog"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed metrics.yaml
var metricsYAML []byte

type metricsConfig struct {
	AllowedMetrics []string `yaml:"allowed_metrics"`
}

var (
	allowedMetrics map[string]bool
	metricsOnce    sync.Once
)

func loadMetrics() {
	var cfg metricsConfig
	if err := yaml.Unmarshal(metricsYAML, &cfg); err != nil {
		slog.Error("failed to parse embedded metrics.yaml", "error", err)
		return
	}

	allowedMetrics = make(map[string]bool, len(cfg.AllowedMetrics))
	for _, name := range cfg.AllowedMetrics {
		allowedMetrics[name] = true
	}

	if len(allowedMetrics) == 0 {
		slog.Error("embedded metrics.yaml contains no allowed_metrics; all metrics will be rejected")
	}
}

func IsAllowedMetric(name string) bool {
	metricsOnce.Do(loadMetrics)
	return allowedMetrics[name]
}
