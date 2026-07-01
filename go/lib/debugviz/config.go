package debugviz

import (
	"fmt"
	"os"
	"strconv"
	"sync"
)

// Config configures the debugviz runtime exporter.
type Config struct {
	ServerURL   string
	ServiceName string
	Enabled     bool
	BatchSize   int
	SampleRate  float64
}

// HTTPMiddlewareConfig configures HTTP root span metadata.
type HTTPMiddlewareConfig struct {
	ServiceName string
}

var (
	globalMu     sync.RWMutex
	globalCfg    Config
	globalExport *exporter
	globalOn     bool
	sourceRoot   string
)

// Configure initializes the runtime exporter. Safe to call once at process start.
func Configure(cfg Config) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if wd, err := os.Getwd(); err == nil {
		sourceRoot = wd
	}

	globalCfg = normalizeConfig(cfg)
	globalOn = resolveEnabled(globalCfg.Enabled)

	if !globalOn {
		globalExport = nil
		return nil
	}

	exp, err := newExporter(globalCfg)
	if err != nil {
		return err
	}
	globalExport = exp
	return nil
}

// ConfigureFromEnv initializes the runtime from DEBUGVIZ_* environment variables.
func ConfigureFromEnv() error {
	enabled := os.Getenv("DEBUGVIZ_ENABLED") == "true"
	if v := os.Getenv("DEBUGVIZ_ENABLED"); v == "1" {
		enabled = true
	}

	cfg := Config{
		ServerURL:   envOr("DEBUGVIZ_SERVER_URL", "http://localhost:4000"),
		ServiceName: os.Getenv("DEBUGVIZ_SERVICE_NAME"),
		Enabled:     enabled,
		BatchSize:   envIntOr("DEBUGVIZ_BATCH_SIZE", 50),
		SampleRate:  envFloatOr("DEBUGVIZ_SAMPLE_RATE", 1.0),
	}
	return Configure(cfg)
}

func normalizeConfig(cfg Config) Config {
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:4000"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = 1.0
	}
	if cfg.SampleRate > 1 {
		cfg.SampleRate = 1.0
	}
	return cfg
}

func resolveEnabled(cfgEnabled bool) bool {
	if cfgEnabled {
		return true
	}
	if v := os.Getenv("DEBUGVIZ_ENABLED"); v == "true" || v == "1" {
		return true
	}
	return compileTimeEnabled()
}

func isEnabled() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalOn
}

func currentExporter() *exporter {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalExport
}

func currentSampleRate() float64 {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalCfg.SampleRate <= 0 {
		return 1.0
	}
	return globalCfg.SampleRate
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloatOr(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func spansEndpoint(serverURL string) (string, error) {
	base := serverURL
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	if base == "" {
		return "", fmt.Errorf("debugviz: empty ServerURL")
	}
	return base + "/api/traces/spans", nil
}
