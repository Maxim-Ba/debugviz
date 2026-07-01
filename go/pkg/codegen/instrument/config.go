package instrument

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const debugvizImportPath = "github.com/Maxim-Ba/debugviz/go/lib/debugviz"

// Config controls which functions receive StartSpan injection.
type Config struct {
	Include          []string `yaml:"include"`
	Exclude          []string `yaml:"exclude"`
	EntryPackages    []string `yaml:"entry_packages"`
	RequireContext   *bool    `yaml:"require_context"`
	AllowNoContext   bool     `yaml:"allow_no_context"`
	DebugvizImport   string   `yaml:"debugviz_import"`
}

// Options configures an instrument run.
type Options struct {
	Dir          string
	Patterns     []string
	ConfigPath   string
	Config       Config
	IncludeTests bool
	DryRun       bool
	Write        bool
}

func (c *Config) requireContext() bool {
	if c.RequireContext == nil {
		return true
	}
	return *c.RequireContext
}

func (c *Config) importPath() string {
	if c.DebugvizImport != "" {
		return c.DebugvizImport
	}
	return debugvizImportPath
}

// LoadConfig reads instrument rules from debugviz.yaml.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var raw struct {
		Include        []string `yaml:"include"`
		Exclude        []string `yaml:"exclude"`
		EntryPackages  []string `yaml:"entry_packages"`
		RequireContext *bool    `yaml:"require_context"`
		AllowNoContext bool     `yaml:"allow_no_context"`
		DebugvizImport string   `yaml:"debugviz_import"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	return Config{
		Include:        raw.Include,
		Exclude:        raw.Exclude,
		EntryPackages:  raw.EntryPackages,
		RequireContext: raw.RequireContext,
		AllowNoContext: raw.AllowNoContext,
		DebugvizImport: raw.DebugvizImport,
	}, nil
}

func resolveConfigPath(dir, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return filepath.Join(dir, "debugviz.yaml")
}
