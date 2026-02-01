package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"gopkg.in/yaml.v3"
)

type metricsConfig struct {
	AllowedMetrics []string `yaml:"allowed_metrics"`
}

var (
	allowedMetrics map[string]bool
	metricsOnce    sync.Once
	metricsErr     error
)

func loadMetrics() {
	_, filename, _, _ := runtime.Caller(0)
	configPath := filepath.Join(filepath.Dir(filename), "..", "..", "..", "config", "metrics.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		metricsErr = err
		return
	}

	var cfg metricsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		metricsErr = err
		return
	}

	allowedMetrics = make(map[string]bool, len(cfg.AllowedMetrics))
	for _, name := range cfg.AllowedMetrics {
		allowedMetrics[name] = true
	}
}

func IsAllowedMetric(name string) bool {
	metricsOnce.Do(loadMetrics)
	if metricsErr != nil {
		return false
	}
	return allowedMetrics[name]
}
