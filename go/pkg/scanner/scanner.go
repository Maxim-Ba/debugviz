package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters"
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

// Options configures a static graph scan.
type Options struct {
	// IncludeTests includes *_test.go files when building file nodes.
	IncludeTests bool
	// Dir is the working directory for go/packages (defaults to os.Getwd).
	Dir string
	// Framework selects entry discoverers (auto, chi, gin, echo, stdlib, grpc, cli, none).
	Framework string
}

// Scan loads Go packages matching patterns and builds a package/file graph
// with import dependency edges.
func Scan(patterns []string, opts Options) (*protocol.Graph, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	patterns = expandPatterns(patterns)

	dir := opts.Dir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		dir = wd
	}

	cfg := &packages.Config{
		Mode: packageLoadMode,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, fmt.Errorf("package load errors (see stderr)")
	}

	rootModule := rootModulePath(pkgs)
	builder := newGraphBuilder(rootModule)

	for _, pkg := range pkgs {
		if pkg == nil || pkg.IllTyped || len(pkg.Errors) > 0 {
			continue
		}
		builder.addPackage(pkg, opts.IncludeTests)
	}

	for _, pkg := range pkgs {
		if pkg == nil || pkg.IllTyped {
			continue
		}
		builder.addImportEdges(pkg)
	}

	scanCtx := adapters.NewScanContext(rootModule, pkgs)
	entries, err := adapters.DiscoverEntries(opts.Framework, dir, scanCtx, pkgs)
	if err != nil {
		return nil, err
	}
	builder.addEntryPoints(entries)
	builder.addCallGraph(pkgs)

	return builder.build(), nil
}

type graphBuilder struct {
	rootModule string
	nodes      map[string]protocol.Node
	edges      map[string]protocol.Edge
}

func newGraphBuilder(rootModule string) *graphBuilder {
	return &graphBuilder{
		rootModule: rootModule,
		nodes:      make(map[string]protocol.Node),
		edges:      make(map[string]protocol.Edge),
	}
}

func (b *graphBuilder) addPackage(pkg *packages.Package, includeTests bool) {
	importPath := pkg.PkgPath
	if importPath == "" {
		return
	}

	pkgID := packageNodeID(importPath)
	if _, ok := b.nodes[pkgID]; !ok {
		b.nodes[pkgID] = protocol.Node{
			ID:   pkgID,
			Type: protocol.NodeTypePackage,
			Name: packageDisplayName(importPath, b.rootModule),
			Path: moduleRelativePath(importPath, b.rootModule),
		}
	}

	files := sourceFiles(pkg, includeTests)
	for _, absPath := range files {
		relPath := toSlashPath(relativeToModule(absPath, pkg, b.rootModule))
		if relPath == "" {
			continue
		}
		fileID := fileNodeID(relPath)
		if _, ok := b.nodes[fileID]; ok {
			continue
		}
		b.nodes[fileID] = protocol.Node{
			ID:   fileID,
			Type: protocol.NodeTypeFile,
			Name: filepath.Base(relPath),
			Path: relPath,
		}
	}
}

func (b *graphBuilder) ensurePackageNode(importPath string) {
	if importPath == "" {
		return
	}
	pkgID := packageNodeID(importPath)
	if _, ok := b.nodes[pkgID]; ok {
		return
	}
	b.nodes[pkgID] = protocol.Node{
		ID:   pkgID,
		Type: protocol.NodeTypePackage,
		Name: packageDisplayName(importPath, b.rootModule),
		Path: moduleRelativePath(importPath, b.rootModule),
	}
}

func (b *graphBuilder) addImportEdges(pkg *packages.Package) {
	sourcePath := pkg.PkgPath
	if sourcePath == "" {
		return
	}
	sourceID := packageNodeID(sourcePath)

	for _, imp := range pkg.Imports {
		if imp == nil {
			continue
		}
		targetPath := imp.PkgPath
		if targetPath == "" {
			continue
		}
		b.ensurePackageNode(targetPath)
		targetID := packageNodeID(targetPath)
		edgeID := importEdgeID(sourcePath, targetPath)
		if _, ok := b.edges[edgeID]; ok {
			continue
		}
		b.edges[edgeID] = protocol.Edge{
			ID:     edgeID,
			Type:   protocol.EdgeTypeImports,
			Source: sourceID,
			Target: targetID,
		}
	}
}

func (b *graphBuilder) addEntryPoints(entries []adapters.EntryPoint) {
	for _, entry := range entries {
		switch entry.Kind {
		case protocol.EntryKindHTTP:
			b.addHTTPEntryPoint(entry)
		case protocol.EntryKindGRPC:
			b.addGRPCEntryPoint(entry)
		}
	}
}

func (b *graphBuilder) addHTTPEntryPoint(entry adapters.EntryPoint) {
	entryID := entryNodeID(entry.Kind, entry.Method, entry.Path)
	b.nodes[entryID] = protocol.Node{
		ID:   entryID,
		Type: protocol.NodeTypeEntryPoint,
		Kind: entry.Kind,
		Name: httpEntryName(entry.Method, entry.Path),
		Metadata: map[string]any{
			"method": entry.Method,
			"path":   entry.Path,
		},
	}

	if entry.HasHandler {
		b.addFunctionNode(entry.Handler)
		handlerID := functionNodeID(entry.Handler.File, entry.Handler.Name)
		edgeID := entryHandlesEdgeID(entryID, handlerID)
		b.edges[edgeID] = protocol.Edge{
			ID:     edgeID,
			Type:   protocol.EdgeTypeEntryHandles,
			Source: entryID,
			Target: handlerID,
		}
	}

	for i, mw := range entry.Middleware {
		b.addMiddlewareNode(mw)
		mwID := middlewareNodeID(mw.File, mw.Name)
		edgeID := middlewareChainEdgeID(entryID, mwID, i)
		b.edges[edgeID] = protocol.Edge{
			ID:     edgeID,
			Type:   protocol.EdgeTypeMiddlewareChain,
			Source: entryID,
			Target: mwID,
			Order:  i,
		}
	}
}

func (b *graphBuilder) addGRPCEntryPoint(entry adapters.EntryPoint) {
	entryID := entryNodeID(entry.Kind, entry.Service, entry.Method)
	b.nodes[entryID] = protocol.Node{
		ID:   entryID,
		Type: protocol.NodeTypeEntryPoint,
		Kind: entry.Kind,
		Name: grpcEntryName(entry.Service, entry.Method),
		Metadata: map[string]any{
			"service": entry.Service,
			"method":  entry.Method,
		},
	}

	if entry.HasHandler {
		b.addFunctionNode(entry.Handler)
		handlerID := functionNodeID(entry.Handler.File, entry.Handler.Name)
		edgeID := entryHandlesEdgeID(entryID, handlerID)
		b.edges[edgeID] = protocol.Edge{
			ID:     edgeID,
			Type:   protocol.EdgeTypeEntryHandles,
			Source: entryID,
			Target: handlerID,
		}
	}
}

func (b *graphBuilder) addFunctionNode(ref adapters.HandlerRef) {
	if ref.File == "" || ref.Name == "" {
		return
	}
	nodeID := functionNodeID(ref.File, ref.Name)
	if _, ok := b.nodes[nodeID]; ok {
		return
	}
	b.nodes[nodeID] = protocol.Node{
		ID:      nodeID,
		Type:    protocol.NodeTypeFunction,
		Name:    ref.Name,
		File:    ref.File,
		Line:    ref.Line,
		Package: ref.Package,
	}
}

func (b *graphBuilder) addMiddlewareNode(ref adapters.HandlerRef) {
	if ref.File == "" || ref.Name == "" {
		return
	}
	nodeID := middlewareNodeID(ref.File, ref.Name)
	if _, ok := b.nodes[nodeID]; ok {
		return
	}
	b.nodes[nodeID] = protocol.Node{
		ID:   nodeID,
		Type: protocol.NodeTypeMiddleware,
		Name: ref.Name,
		File: ref.File,
	}
}

func (b *graphBuilder) build() *protocol.Graph {
	nodes := make([]protocol.Node, 0, len(b.nodes))
	for _, node := range b.nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type < nodes[j].Type
		}
		return nodes[i].ID < nodes[j].ID
	})

	edges := make([]protocol.Edge, 0, len(b.edges))
	for _, edge := range b.edges {
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})

	return &protocol.Graph{
		Version:     protocol.GraphVersion,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		RootModule:  b.rootModule,
		Nodes:       nodes,
		Edges:       edges,
	}
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
	seen := make(map[string]struct{}, len(pkg.GoFiles)+len(pkg.CompiledGoFiles))
	var files []string

	add := func(path string) {
		if path == "" {
			return
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}

	for _, path := range pkg.CompiledGoFiles {
		add(path)
	}
	for _, path := range pkg.GoFiles {
		add(path)
	}
	sort.Strings(files)
	return files
}

func moduleRelativePath(importPath, rootModule string) string {
	if rootModule == "" {
		return importPath
	}
	prefix := rootModule + "/"
	if strings.HasPrefix(importPath, prefix) {
		return strings.TrimPrefix(importPath, prefix)
	}
	return importPath
}

func packageDisplayName(importPath, rootModule string) string {
	rel := moduleRelativePath(importPath, rootModule)
	if rel == "" {
		return importPath
	}
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		return rel[idx+1:]
	}
	return rel
}

func relativeToModule(absPath string, pkg *packages.Package, rootModule string) string {
	absPath = filepath.Clean(absPath)
	if pkg.Module != nil && pkg.Module.Dir != "" {
		moduleDir := filepath.Clean(pkg.Module.Dir)
		if rel, err := filepath.Rel(moduleDir, absPath); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	if rootModule != "" {
		suffix := moduleRelativePath(pkg.PkgPath, rootModule)
		if suffix != "" {
			return filepath.Join(suffix, filepath.Base(absPath))
		}
	}
	return absPath
}

func toSlashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
}

func packageNodeID(importPath string) string {
	return "pkg:" + importPath
}

func fileNodeID(relPath string) string {
	return "file:" + relPath
}

func importEdgeID(sourcePath, targetPath string) string {
	return "edge:imports:" + slugImportPath(sourcePath) + ":" + slugImportPath(targetPath)
}

func slugImportPath(importPath string) string {
	slug := strings.NewReplacer("/", "-", ".", "-").Replace(importPath)
	return slug
}

func expandPatterns(patterns []string) []string {
	expanded := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.Contains(pattern, "...") {
			expanded = append(expanded, pattern)
			continue
		}
		if strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "../") || filepath.IsAbs(pattern) {
			expanded = append(expanded, filepath.ToSlash(pattern)+"/...")
			continue
		}
		expanded = append(expanded, pattern)
	}
	return expanded
}
