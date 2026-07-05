package wire

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const packageLoadMode = 0 // wire reads files directly, no go/packages load required for MVP

// Result summarizes one wired file.
type Result struct {
	Path    string
	Before  []byte
	After   []byte
	Changed bool
	Plan    wirePlan
}

// Run wires entry hooks into Go main and router files.
func Run(opts Options) ([]Result, error) {
	if len(opts.Patterns) == 0 {
		opts.Patterns = []string{"./..."}
	}

	dir := opts.Dir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		dir = wd
	}

	cfg := opts.Config
	configPath := resolveConfigPath(dir, opts.ConfigPath)
	configDir := dir
	if opts.ConfigPath != "" {
		if filepath.IsAbs(opts.ConfigPath) {
			configDir = filepath.Dir(opts.ConfigPath)
		} else {
			configDir = filepath.Dir(filepath.Join(dir, opts.ConfigPath))
		}
	} else if _, err := os.Stat(configPath); err == nil {
		configDir = filepath.Dir(configPath)
	}

	if opts.ConfigPath != "" || cfg.ServiceName == "" {
		loaded, err := LoadConfig(configPath)
		if err != nil {
			return nil, err
		}
		if cfg.ServiceName == "" {
			cfg.ServiceName = loaded.ServiceName
		}
		if cfg.ServerURL == "" {
			cfg.ServerURL = loaded.ServerURL
		}
		if cfg.Wire.Main == "" {
			cfg.Wire = loaded.Wire
		}
		if cfg.DebugvizImport == "" {
			cfg.DebugvizImport = loaded.DebugvizImport
		}
	}

	targets, err := resolveTargets(dir, configDir, cfg, opts.Patterns, opts.IncludeTests)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no wire targets found (set wire.main in debugviz.yaml)")
	}

	var results []Result
	for _, target := range targets {
		src, err := os.ReadFile(target.absPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", target.relPath, err)
		}

		out, plan, err := rewriteFile(src, target.absPath, cfg, target.annotation, target.isMain)
		if err != nil {
			return nil, fmt.Errorf("wire %s: %w", target.relPath, err)
		}
		if !planHasChanges(plan) {
			continue
		}

		result := Result{
			Path:    target.relPath,
			Before:  src,
			After:   out,
			Changed: true,
			Plan:    plan,
		}
		results = append(results, result)

		if opts.DryRun {
			continue
		}
		if opts.Write {
			if err := os.WriteFile(target.absPath, out, 0o644); err != nil {
				return nil, fmt.Errorf("write %s: %w", target.relPath, err)
			}
		}
	}

	return results, nil
}

type wireTarget struct {
	absPath    string
	relPath    string
	isMain     bool
	annotation *AppAnnotation
}

func resolveTargets(dir, configDir string, cfg Config, patterns []string, includeTests bool) ([]wireTarget, error) {
	var targets []wireTarget
	seen := make(map[string]struct{})

	addTarget := func(absPath, relPath string, isMain bool) error {
		absPath = filepath.Clean(absPath)
		relPath = filepath.ToSlash(relPath)
		if !includeTests && strings.HasSuffix(relPath, "_test.go") {
			return nil
		}
		if _, ok := seen[absPath]; ok {
			return nil
		}
		seen[absPath] = struct{}{}

		src, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		ann, err := parseAppAnnotationFromSource(src, absPath)
		if err != nil {
			return err
		}
		targets = append(targets, wireTarget{
			absPath:    absPath,
			relPath:    relPath,
			isMain:     isMain,
			annotation: ann,
		})
		return nil
	}

	resolvePath := func(path string) string {
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(configDir, path)
	}

	if cfg.Wire.Main != "" {
		mainPath := resolvePath(cfg.Wire.Main)
		rel, err := filepath.Rel(dir, mainPath)
		if err != nil {
			rel = mainPath
		}
		if err := addTarget(mainPath, rel, true); err != nil {
			return nil, err
		}
	}

	for _, relRouter := range cfg.Wire.HTTP.RouterFiles {
		routerPath := resolvePath(relRouter)
		rel, err := filepath.Rel(dir, routerPath)
		if err != nil {
			rel = routerPath
		}
		if err := addTarget(routerPath, rel, false); err != nil {
			return nil, err
		}
	}

	if len(targets) == 0 {
		for _, pattern := range expandPatterns(patterns) {
			matches, err := filepath.Glob(filepath.Join(dir, pattern, "main.go"))
			if err != nil {
				return nil, err
			}
			for _, match := range matches {
				rel, err := filepath.Rel(dir, match)
				if err != nil {
					rel = match
				}
				if err := addTarget(match, rel, true); err != nil {
					return nil, err
				}
			}
		}
	}

	if cfg.httpEnabled() && len(cfg.Wire.HTTP.RouterFiles) == 0 {
		routerFiles, err := discoverChiRouters(dir, patterns, includeTests)
		if err != nil {
			return nil, err
		}
		for _, rf := range routerFiles {
			if err := addTarget(rf.absPath, rf.relPath, false); err != nil {
				return nil, err
			}
		}
	}

	return targets, nil
}

func discoverChiRouters(dir string, patterns []string, includeTests bool) ([]wireTarget, error) {
	var routers []wireTarget
	seen := make(map[string]struct{})

	for _, pattern := range expandPatterns(patterns) {
		root := filepath.Join(dir, pattern)
		if strings.HasSuffix(pattern, "...") {
			root = filepath.Join(dir, strings.TrimSuffix(pattern, "..."))
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if !includeTests && strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if strings.HasSuffix(filepath.Base(path), "main.go") {
				return nil
			}
			if _, ok := seen[path]; ok {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !strings.Contains(string(src), "chi.NewRouter") {
				return nil
			}
			seen[path] = struct{}{}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				rel = path
			}
			routers = append(routers, wireTarget{absPath: path, relPath: filepath.ToSlash(rel)})
			return nil
		})
	}

	return routers, nil
}

func expandPatterns(patterns []string) []string {
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"./..."}
	}
	return out
}
