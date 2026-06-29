package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

type manualConfig struct {
	Entries []manualEntry `yaml:"entries"`
}

type manualEntry struct {
	Kind    string `yaml:"kind"`
	Method  string `yaml:"method"`
	Path    string `yaml:"path"`
	Handler string `yaml:"handler"`
	Service string `yaml:"service"`
	Command string `yaml:"command"`
}

// LoadManualEntries reads fallback entry points from debugviz.yaml.
func LoadManualEntries(dir string, ctx *ScanContext) ([]EntryPoint, error) {
	if dir == "" {
		return nil, nil
	}

	path := filepath.Join(dir, "debugviz.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read debugviz.yaml: %w", err)
	}

	var cfg manualConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse debugviz.yaml: %w", err)
	}

	var entries []EntryPoint
	for _, item := range cfg.Entries {
		entry, ok := manualEntryToPoint(item, ctx)
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func manualEntryToPoint(item manualEntry, ctx *ScanContext) (EntryPoint, bool) {
	kind := protocol.EntryKind(strings.ToLower(strings.TrimSpace(item.Kind)))
	switch kind {
	case protocol.EntryKindHTTP:
		method := strings.ToUpper(strings.TrimSpace(item.Method))
		path := strings.TrimSpace(item.Path)
		if method == "" || path == "" {
			return EntryPoint{}, false
		}
		entry := EntryPoint{
			Kind:   kind,
			Method: method,
			Path:   path,
		}
		if item.Handler != "" {
			if ref, ok := ctx.FindFuncByName(item.Handler); ok {
				entry.Handler = ref
				entry.HasHandler = true
			}
		}
		return entry, true
	case protocol.EntryKindGRPC:
		service := strings.TrimSpace(item.Service)
		method := strings.TrimSpace(item.Method)
		if service == "" || method == "" {
			return EntryPoint{}, false
		}
		entry := EntryPoint{
			Kind:    kind,
			Service: service,
			Method:  method,
		}
		if item.Handler != "" {
			if ref, ok := ctx.FindFuncByName(item.Handler); ok {
				entry.Handler = ref
				entry.HasHandler = true
			}
		}
		return entry, true
	case protocol.EntryKindCLI:
		command := strings.TrimSpace(item.Command)
		if command == "" {
			return EntryPoint{}, false
		}
		entry := EntryPoint{
			Kind:    kind,
			Command: command,
		}
		if item.Handler != "" {
			if ref, ok := ctx.FindFuncByName(item.Handler); ok {
				entry.Handler = ref
				entry.HasHandler = true
			}
		}
		return entry, true
	default:
		return EntryPoint{}, false
	}
}
