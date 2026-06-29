package adapters

import (
	"strings"

	"golang.org/x/tools/go/packages"
)

// Framework names for scanner CLI (--framework).
const (
	FrameworkAuto    = "auto"
	FrameworkChi     = "chi"
	FrameworkGin     = "gin"
	FrameworkEcho    = "echo"
	FrameworkStdlib  = "stdlib"
	FrameworkNone    = "none"
)

// SelectDiscoverers returns entry discoverers for the requested framework mode.
func SelectDiscoverers(framework string, pkgs []*packages.Package) []EntryDiscoverer {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "", FrameworkAuto:
		return autoDiscoverers(pkgs)
	case FrameworkChi:
		return []EntryDiscoverer{NewChi()}
	case FrameworkGin:
		return []EntryDiscoverer{NewGin()}
	case FrameworkEcho:
		return []EntryDiscoverer{NewEcho()}
	case FrameworkStdlib:
		return []EntryDiscoverer{NewStdlib()}
	case FrameworkNone:
		return nil
	default:
		return autoDiscoverers(pkgs)
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
	var all []EntryPoint
	for _, discoverer := range SelectDiscoverers(framework, pkgs) {
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
		key := string(entry.Kind) + "|" + entry.Method + "|" + entry.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	return out
}
