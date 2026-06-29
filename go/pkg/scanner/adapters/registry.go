package adapters

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// Framework names for scanner CLI (--framework).
const (
	FrameworkAuto   = "auto"
	FrameworkChi    = "chi"
	FrameworkGin    = "gin"
	FrameworkEcho   = "echo"
	FrameworkStdlib = "stdlib"
	FrameworkGRPC   = "grpc"
	FrameworkCLI    = "cli"
	FrameworkNone   = "none"
)

// NormalizeFramework validates and canonicalizes a --framework flag value.
func NormalizeFramework(framework string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "", FrameworkAuto:
		return FrameworkAuto, nil
	case FrameworkChi, FrameworkGin, FrameworkEcho, FrameworkStdlib, FrameworkGRPC, FrameworkCLI, FrameworkNone:
		return strings.ToLower(strings.TrimSpace(framework)), nil
	default:
		return "", fmt.Errorf("unknown framework %q: want auto, chi, gin, echo, stdlib, grpc, cli", framework)
	}
}

// SelectDiscoverers returns entry discoverers for the requested framework mode.
func SelectDiscoverers(framework string, pkgs []*packages.Package) ([]EntryDiscoverer, error) {
	mode, err := NormalizeFramework(framework)
	if err != nil {
		return nil, err
	}

	switch mode {
	case FrameworkAuto:
		return autoDiscoverers(pkgs), nil
	case FrameworkChi:
		return []EntryDiscoverer{NewChi()}, nil
	case FrameworkGin:
		return []EntryDiscoverer{NewGin()}, nil
	case FrameworkEcho:
		return []EntryDiscoverer{NewEcho()}, nil
	case FrameworkStdlib:
		return []EntryDiscoverer{NewStdlib()}, nil
	case FrameworkGRPC:
		return []EntryDiscoverer{NewGRPC()}, nil
	case FrameworkCLI:
		// CLI discoverer is implemented in issue 1.6.
		return nil, nil
	case FrameworkNone:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown framework %q", framework)
	}
}

func autoDiscoverers(pkgs []*packages.Package) []EntryDiscoverer {
	var discoverers []EntryDiscoverer
	if pkgImports(pkgs, isChiImport) {
		discoverers = append(discoverers, NewChi())
	}
	if pkgImports(pkgs, isGinImport) {
		discoverers = append(discoverers, NewGin())
	}
	if pkgImports(pkgs, isEchoImport) {
		discoverers = append(discoverers, NewEcho())
	}
	if pkgImports(pkgs, isGRPCImport) {
		discoverers = append(discoverers, NewGRPC())
	}
	discoverers = append(discoverers, NewStdlib())
	return discoverers
}

// DiscoverEntries runs discoverers and merges manual debugviz.yaml entries.
func DiscoverEntries(framework, configDir string, ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	entries, err := discoverWithAdapters(framework, ctx, pkgs)
	if err != nil {
		return nil, err
	}

	manual, err := LoadManualEntries(configDir, ctx)
	if err != nil {
		return nil, err
	}
	return dedupeEntries(append(entries, manual...)), nil
}

func discoverWithAdapters(framework string, ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	discoverers, err := SelectDiscoverers(framework, pkgs)
	if err != nil {
		return nil, err
	}

	var all []EntryPoint
	for _, discoverer := range discoverers {
		found, err := discoverer.Discover(ctx, pkgs)
		if err != nil {
			return nil, err
		}
		all = append(all, found...)
	}
	return dedupeEntries(all), nil
}

func dedupeEntries(entries []EntryPoint) []EntryPoint {
	seen := make(map[string]struct{}, len(entries))
	out := make([]EntryPoint, 0, len(entries))
	for _, entry := range entries {
		key := entryDedupeKey(entry)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	return out
}

func entryDedupeKey(entry EntryPoint) string {
	switch entry.Kind {
	case protocol.EntryKindGRPC:
		return string(entry.Kind) + "|" + entry.Service + "|" + entry.Method
	default:
		return string(entry.Kind) + "|" + entry.Method + "|" + entry.Path
	}
}
