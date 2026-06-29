package scanner

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

type funcRef struct {
	File    string
	Line    int
	Name    string
	Package string
}

type callRef struct {
	Caller     funcRef
	Callee     funcRef
	Confidence protocol.CallConfidence
}

type callInspector struct {
	pkg        *packages.Package
	pkgIndex   map[string]*packages.Package
	rootModule string
	calls      []callRef
	stack      []*ast.FuncDecl
}

func buildCallGraph(pkgs []*packages.Package, rootModule string) []callRef {
	pkgIndex := make(map[string]*packages.Package, len(pkgs))
	for _, pkg := range pkgs {
		if pkg != nil {
			pkgIndex[pkg.PkgPath] = pkg
		}
	}

	var calls []callRef
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil || pkg.TypesInfo == nil {
			continue
		}
		if rootModule != "" && !strings.HasPrefix(pkg.PkgPath, rootModule) {
			continue
		}
		for _, file := range pkg.Syntax {
			inspector := &callInspector{
				pkg:        pkg,
				pkgIndex:   pkgIndex,
				rootModule: rootModule,
			}
			ast.Walk(inspector, file)
			calls = append(calls, inspector.calls...)
		}
	}
	return calls
}

type funcBodyInspector struct {
	parent *callInspector
}

func (i *callInspector) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		i.stack = append(i.stack, n)
		return &funcBodyInspector{parent: i}
	case *ast.CallExpr:
		i.recordCall(n)
	}

	return i
}

func (f *funcBodyInspector) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		if len(f.parent.stack) > 0 {
			f.parent.stack = f.parent.stack[:len(f.parent.stack)-1]
		}
		return nil
	}
	if call, ok := node.(*ast.CallExpr); ok {
		f.parent.recordCall(call)
	}
	return f.parent
}

func (i *callInspector) recordCall(call *ast.CallExpr) {
	if len(i.stack) == 0 {
		return
	}

	callerDecl := i.stack[len(i.stack)-1]
	caller, ok := i.funcRefFromDecl(callerDecl)
	if !ok {
		return
	}

	calleeObj, ok := i.resolveCallee(call)
	if !ok {
		return
	}
	callee, ok := i.funcRefFromObject(calleeObj)
	if !ok {
		return
	}
	if i.rootModule != "" && !strings.HasPrefix(callee.Package, i.rootModule) {
		return
	}

	i.calls = append(i.calls, callRef{
		Caller:     caller,
		Callee:     callee,
		Confidence: callConfidence(i.pkg.TypesInfo, call),
	})
}

func (i *callInspector) resolveCallee(call *ast.CallExpr) (*types.Func, bool) {
	info := i.pkg.TypesInfo
	if info == nil {
		return nil, false
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj := info.ObjectOf(fun)
		fn, ok := obj.(*types.Func)
		return fn, ok && fn.Pkg() != nil && fun.Name != "init"
	case *ast.SelectorExpr:
		if sel, ok := info.Selections[fun]; ok {
			fn, ok := sel.Obj().(*types.Func)
			return fn, ok
		}
		obj := info.ObjectOf(fun.Sel)
		fn, ok := obj.(*types.Func)
		return fn, ok
	default:
		return nil, false
	}
}

func callConfidence(info *types.Info, call *ast.CallExpr) protocol.CallConfidence {
	if info == nil {
		return protocol.CallConfidenceUnknown
	}

	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if sel, ok := info.Selections[fun]; ok {
			if recv := sel.Recv(); recv != nil && types.IsInterface(recv) {
				return protocol.CallConfidenceInterface
			}
			return protocol.CallConfidenceStatic
		}
		return protocol.CallConfidenceUnknown
	case *ast.Ident:
		return protocol.CallConfidenceStatic
	default:
		return protocol.CallConfidenceUnknown
	}
}

func (i *callInspector) funcRefFromDecl(fn *ast.FuncDecl) (funcRef, bool) {
	if fn == nil || fn.Name == nil || fn.Name.Name == "" {
		return funcRef{}, false
	}
	pos := i.pkg.Fset.Position(fn.Pos())
	return funcRef{
		File:    relFilePath(pos.Filename, i.pkg, i.rootModule),
		Line:    pos.Line,
		Name:    fn.Name.Name,
		Package: i.pkg.PkgPath,
	}, true
}

func (i *callInspector) funcRefFromObject(fn *types.Func) (funcRef, bool) {
	if fn == nil || fn.Pkg() == nil {
		return funcRef{}, false
	}

	pos := i.pkg.Fset.Position(fn.Pos())
	owner := i.pkgIndex[fn.Pkg().Path()]
	if owner == nil {
		owner = i.pkg
	}

	return funcRef{
		File:    relFilePath(pos.Filename, owner, i.rootModule),
		Line:    pos.Line,
		Name:    fn.Name(),
		Package: fn.Pkg().Path(),
	}, true
}

func relFilePath(absPath string, pkg *packages.Package, rootModule string) string {
	absPath = filepath.Clean(absPath)
	if pkg != nil && pkg.Module != nil && pkg.Module.Dir != "" {
		moduleDir := filepath.Clean(pkg.Module.Dir)
		if rel, err := filepath.Rel(moduleDir, absPath); err == nil && !strings.HasPrefix(rel, "..") {
			return toSlashPath(rel)
		}
	}
	return toSlashPath(absPath)
}

func (b *graphBuilder) addCallGraph(pkgs []*packages.Package) {
	for _, ref := range buildCallGraph(pkgs, b.rootModule) {
		b.addFunctionNodeRef(ref.Caller)
		b.addFunctionNodeRef(ref.Callee)

		sourceID := functionNodeID(ref.Caller.File, ref.Caller.Name)
		targetID := functionNodeID(ref.Callee.File, ref.Callee.Name)
		edgeID := callsEdgeID(sourceID, targetID)
		if _, exists := b.edges[edgeID]; exists {
			continue
		}
		b.edges[edgeID] = protocol.Edge{
			ID:         edgeID,
			Type:       protocol.EdgeTypeCalls,
			Source:     sourceID,
			Target:     targetID,
			Confidence: ref.Confidence,
		}
	}
}

func (b *graphBuilder) addFunctionNodeRef(ref funcRef) {
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

func callsEdgeID(sourceID, targetID string) string {
	return "edge:calls:" + slugImportPath(sourceID) + ":" + slugImportPath(targetID)
}
