package wire

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const debugvizImportPath = "github.com/Maxim-Ba/debugviz/go/lib/debugviz"

// HTTPWireConfig controls HTTP entry hook injection.
type HTTPWireConfig struct {
	ListenAndServe bool     `yaml:"listen_and_serve"`
	RouterFiles    []string `yaml:"router_files"`
}

// GRPCWireConfig controls gRPC interceptor injection.
type GRPCWireConfig struct {
	NewServer bool `yaml:"new_server"`
}

// CLIWireConfig controls cobra / urfave CLI wiring.
type CLIWireConfig struct {
	CobraExecute bool `yaml:"cobra_execute"`
}

// WorkerJobTarget names a worker handler call to wrap with RunJob.
type WorkerJobTarget struct {
	Name string `yaml:"name"`
	File string `yaml:"file"`
}

// WorkerWireConfig controls worker job span injection.
type WorkerWireConfig struct {
	Targets []WorkerJobTarget `yaml:"targets"`
}

// WireConfig selects files and entry hook strategies.
type WireConfig struct {
	Main   string           `yaml:"main"`
	HTTP   HTTPWireConfig   `yaml:"http"`
	GRPC   GRPCWireConfig   `yaml:"grpc"`
	CLI    CLIWireConfig    `yaml:"cli"`
	Worker WorkerWireConfig `yaml:"worker"`
}

// Config is the wire section of debugviz.yaml plus shared fields.
type Config struct {
	ServerURL      string     `yaml:"server_url"`
	ServiceName    string     `yaml:"service_name"`
	Wire           WireConfig `yaml:"wire"`
	DebugvizImport string     `yaml:"debugviz_import"`
}

// Options configures a wire run.
type Options struct {
	Dir          string
	Patterns     []string
	ConfigPath   string
	Config       Config
	IncludeTests bool
	DryRun       bool
	Write        bool
}

func (c *Config) importPath() string {
	if c.DebugvizImport != "" {
		return c.DebugvizImport
	}
	return debugvizImportPath
}

// LoadConfig reads debugviz.yaml wire settings.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func resolveConfigPath(dir, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return filepath.Join(dir, "debugviz.yaml")
}

func (c *Config) httpEnabled() bool {
	return c.Wire.HTTP.ListenAndServe || len(c.Wire.HTTP.RouterFiles) > 0
}

func (c *Config) grpcEnabled() bool {
	return c.Wire.GRPC.NewServer
}

func (c *Config) cliEnabled() bool {
	return c.Wire.CLI.CobraExecute
}

func (c *Config) workerEnabled() bool {
	return len(c.Wire.Worker.Targets) > 0
}
