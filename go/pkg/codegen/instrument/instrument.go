package instrument

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

const packageLoadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedModule |
	packages.NeedDeps |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo

// Result summarizes one instrumented file.
type Result struct {
	Path    string
	Changed bool
	Funcs   int
}

// Run instruments Go packages matching patterns.
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
	if opts.ConfigPath != "" {
		loaded, err := LoadConfig(opts.ConfigPath)
		if err != nil {
			return nil, err
		}
		cfg = loaded
	} else {
		configPath := resolveConfigPath(dir, "")
		loaded, err := LoadConfig(configPath)
		if err != nil {
			return nil, err
		}
		cfg = loaded
	}

	cfgPatterns := expandPatterns(opts.Patterns)
	pkgs, err := packages.Load(&packages.Config{
		Mode: packageLoadMode,
		Dir:  dir,
	}, cfgPatterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package load errors (see stderr)")
	}

	rootModule := rootModulePath(pkgs)
	var results []Result

	for _, pkg := range pkgs {
		if pkg == nil || pkg.IllTyped || len(pkg.Errors) > 0 {
			continue
		}
		if pkg.Types == nil || pkg.Fset == nil {
			continue
		}

		files := sourceFiles(pkg, opts.IncludeTests)
		for _, absPath := range files {
			relPath := moduleRelativeFile(absPath, pkg, rootModule)
			if relPath == "" {
				continue
			}
			if !shouldInstrumentFile(relPath, cfg, opts.IncludeTests) {
				continue
			}

			file := fileForPath(pkg, absPath)
			if file == nil {
				continue
			}

			candidates := analyzeFile(file, pkg.Types, pkg.TypesInfo, cfg)
			if len(candidates) == 0 {
				continue
			}

			updated, err := injectCandidates(file, pkg.Fset, candidates, cfg)
			if err != nil {
				return nil, fmt.Errorf("inject %s: %w", relPath, err)
			}
			if updated == nil {
				continue
			}

			result := Result{
				Path:    relPath,
				Changed: true,
				Funcs:   len(candidates),
			}
			results = append(results, result)

			if opts.DryRun {
				continue
			}
			if opts.Write {
				if err := os.WriteFile(absPath, updated, 0o644); err != nil {
					return nil, fmt.Errorf("write %s: %w", absPath, err)
				}
			}
		}
	}

	return results, nil
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

func rootModulePath(pkgs []*packages.Package) string {
	for _, pkg := range pkgs {
		if pkg != nil && pkg.Module != nil && pkg.Module.Path != "" {
			return pkg.Module.Path
		}
	}
	return ""
}

func sourceFiles(pkg *packages.Package, includeTests bool) []string {
	seen := make(map[string]struct{})
	var files []string
	for _, name := range pkg.CompiledGoFiles {
		if name == "" {
			continue
		}
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		files = append(files, name)
	}
	if len(files) > 0 {
		return files
	}
	for _, name := range pkg.GoFiles {
		if name == "" {
			continue
		}
		if !includeTests && strings.HasSuffix(name, "_test.go") {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		files = append(files, name)
	}
	return files
}

func fileForPath(pkg *packages.Package, absPath string) *ast.File {
	if pkg == nil {
		return nil
	}
	absPath = filepath.Clean(absPath)
	for i, name := range pkg.CompiledGoFiles {
		if filepath.Clean(name) == absPath && i < len(pkg.Syntax) {
			return pkg.Syntax[i]
		}
	}
	return nil
}

func moduleRelativeFile(absPath string, pkg *packages.Package, rootModule string) string {
	if pkg != nil && pkg.Module != nil && pkg.Module.Dir != "" {
		rel, err := filepath.Rel(pkg.Module.Dir, absPath)
		if err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(absPath)
}

// InstrumentFile instruments a single Go source file. Used by golden tests.
func InstrumentFile(src []byte, filename string, cfg Config, pkgName string) ([]byte, int, error) {
	fset, file, err := parseSource(filename, src)
	if err != nil {
		return nil, 0, err
	}

	pkg, info := typeCheckFile(fset, file, pkgName)
	candidates := analyzeFile(file, pkg, info, cfg)
	if len(candidates) == 0 {
		return src, 0, nil
	}

	updated, err := injectCandidates(file, fset, candidates, cfg)
	if err != nil {
		return nil, 0, err
	}
	return updated, len(candidates), nil
}
