package adapters

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// Chi discoverer finds HTTP routes registered via go-chi/chi.
type Chi struct{}

func NewChi() *Chi { return &Chi{} }

func (c *Chi) Name() string { return "chi" }

func (c *Chi) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	if !pkgImports(pkgs, isChiImport) {
		return nil, nil
	}

	var entries []EntryPoint
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		if !pkgImports([]*packages.Package{pkg}, isChiImport) {
			continue
		}
		for _, file := range pkg.Syntax {
			w := &routeWalker{ctx: ctx, pkg: pkg}
			w.walkFile(file)
			entries = append(entries, w.entries...)
		}
	}
	return entries, nil
}

// Gin discoverer finds HTTP routes registered via gin-gonic/gin.
type Gin struct{}

func NewGin() *Gin { return &Gin{} }

func (g *Gin) Name() string { return "gin" }

func (g *Gin) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	if !pkgImports(pkgs, isGinImport) {
		return nil, nil
	}

	var entries []EntryPoint
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		if !pkgImports([]*packages.Package{pkg}, isGinImport) {
			continue
		}
		for _, file := range pkg.Syntax {
			w := &ginWalker{routeWalker: routeWalker{ctx: ctx, pkg: pkg}}
			w.walkFile(file)
			entries = append(entries, w.entries...)
		}
	}
	return entries, nil
}

type ginWalker struct {
	routeWalker
}

func (w *ginWalker) walkFile(file *ast.File) {
	w.routerVars = make(map[string]routerFrame)
	ast.Walk(w, file)
}

func (w *ginWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		return w.visitFuncDecl(n)
	case *ast.AssignStmt:
		w.trackGinAssignments(n)
	case *ast.ExprStmt:
		w.trackGinCall(n.X)
	case *ast.BlockStmt:
		for _, stmt := range n.List {
			if es, ok := stmt.(*ast.ExprStmt); ok {
				w.trackGinCall(es.X)
			}
		}
	}

	return w
}

func (w *ginWalker) visitFuncDecl(fn *ast.FuncDecl) ast.Visitor {
	if fn.Body == nil {
		return w
	}

	saved := cloneRouterVars(w.routerVars)
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			w.routerVars[name.Name] = routerFrame{}
		}
	}

	ast.Walk(w, fn.Body)

	w.routerVars = saved
	return nil
}

func (w *ginWalker) trackGinAssignments(stmt *ast.AssignStmt) {
	if len(stmt.Rhs) != 1 {
		return
	}
	call, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Group" {
		if recv, ok := sel.X.(*ast.Ident); ok {
			if parent, ok := w.frameForVar(recv.Name); ok && len(call.Args) > 0 {
				if path, ok := stringLiteral(call.Args[0]); ok {
					child := routerFrame{
						prefix:     joinRoutePath(parent.prefix, path),
						middleware: append([]HandlerRef{}, parent.middleware...),
					}
					for _, lhs := range stmt.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok {
							w.routerVars[ident.Name] = child
						}
					}
				}
			}
		}
		return
	}

	if !isNamedCall(call, "Default") && !isNamedCall(call, "New") {
		return
	}
	for _, lhs := range stmt.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			w.routerVars[ident.Name] = routerFrame{}
		}
	}
}

func (w *ginWalker) trackGinCall(expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	frame, ok := w.frameForVar(recv.Name)
	if !ok {
		return
	}

	switch sel.Sel.Name {
	case "Use":
		w.trackMiddleware(recv.Name, frame, call.Args)
	default:
		w.trackHTTPMethod(recv.Name, frame, sel.Sel.Name, call.Args)
	}
}

func (w *ginWalker) frameForVar(name string) (routerFrame, bool) {
	frame, ok := w.routerVars[name]
	return frame, ok
}

// Echo discoverer finds HTTP routes registered via labstack/echo.
type Echo struct{}

func NewEcho() *Echo { return &Echo{} }

func (e *Echo) Name() string { return "echo" }

func (e *Echo) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	if !pkgImports(pkgs, isEchoImport) {
		return nil, nil
	}

	var entries []EntryPoint
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		if !pkgImports([]*packages.Package{pkg}, isEchoImport) {
			continue
		}
		for _, file := range pkg.Syntax {
			w := &echoWalker{routeWalker: routeWalker{ctx: ctx, pkg: pkg}}
			w.walkFile(file)
			entries = append(entries, w.entries...)
		}
	}
	return entries, nil
}

type echoWalker struct {
	routeWalker
}

func (w *echoWalker) walkFile(file *ast.File) {
	w.routerVars = make(map[string]routerFrame)
	ast.Walk(w, file)
}

func (w *echoWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		return w.visitFuncDecl(n)
	case *ast.AssignStmt:
		w.trackEchoAssignments(n)
	case *ast.ExprStmt:
		w.trackEchoCall(n.X)
	case *ast.BlockStmt:
		for _, stmt := range n.List {
			if es, ok := stmt.(*ast.ExprStmt); ok {
				w.trackEchoCall(es.X)
			}
		}
	}

	return w
}

func (w *echoWalker) visitFuncDecl(fn *ast.FuncDecl) ast.Visitor {
	if fn.Body == nil {
		return w
	}

	saved := cloneRouterVars(w.routerVars)
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			w.routerVars[name.Name] = routerFrame{}
		}
	}

	ast.Walk(w, fn.Body)

	w.routerVars = saved
	return nil
}

func (w *echoWalker) trackEchoAssignments(stmt *ast.AssignStmt) {
	if len(stmt.Rhs) != 1 {
		return
	}
	call, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Group" {
		if recv, ok := sel.X.(*ast.Ident); ok {
			if parent, ok := w.frameForVar(recv.Name); ok && len(call.Args) > 0 {
				if path, ok := stringLiteral(call.Args[0]); ok {
					child := routerFrame{
						prefix:     joinRoutePath(parent.prefix, path),
						middleware: append([]HandlerRef{}, parent.middleware...),
					}
					for _, lhs := range stmt.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok {
							w.routerVars[ident.Name] = child
						}
					}
				}
			}
		}
		return
	}

	if !isNamedCall(call, "New") {
		return
	}
	for _, lhs := range stmt.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			w.routerVars[ident.Name] = routerFrame{}
		}
	}
}

func (w *echoWalker) trackEchoCall(expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	recv, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	frame, ok := w.frameForVar(recv.Name)
	if !ok {
		return
	}

	switch sel.Sel.Name {
	case "Use":
		w.trackMiddleware(recv.Name, frame, call.Args)
	default:
		w.trackHTTPMethod(recv.Name, frame, sel.Sel.Name, call.Args)
	}
}

func (w *echoWalker) frameForVar(name string) (routerFrame, bool) {
	frame, ok := w.routerVars[name]
	return frame, ok
}

// Stdlib discoverer finds routes registered via net/http.
type Stdlib struct{}

func NewStdlib() *Stdlib { return &Stdlib{} }

func (s *Stdlib) Name() string { return "stdlib" }

func (s *Stdlib) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	var entries []EntryPoint
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			w := &stdlibWalker{ctx: ctx, pkg: pkg}
			w.walkFile(file)
			entries = append(entries, w.entries...)
		}
	}
	return entries, nil
}

type stdlibWalker struct {
	ctx       *ScanContext
	pkg       *packages.Package
	entries   []EntryPoint
	muxVars   map[string]struct{}
}

func (w *stdlibWalker) walkFile(file *ast.File) {
	w.muxVars = make(map[string]struct{})
	ast.Walk(w, file)
}

func (w *stdlibWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.AssignStmt:
		if len(n.Rhs) == 1 {
			if call, ok := n.Rhs[0].(*ast.CallExpr); ok && isNamedCall(call, "NewServeMux") {
				for _, lhs := range n.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						w.muxVars[ident.Name] = struct{}{}
					}
				}
			}
		}
	case *ast.ExprStmt:
		w.trackStdlibCall(n.X, "")
	}

	return w
}

func (w *stdlibWalker) trackStdlibCall(expr ast.Expr, prefix string) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}

	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if fun.Sel.Name != "HandleFunc" && fun.Sel.Name != "Handle" {
			return
		}
		recv, ok := fun.X.(*ast.Ident)
		if !ok {
			return
		}
		if recv.Name == "http" {
			w.addStdlibRoute("", call.Args)
			return
		}
		if _, ok := w.muxVars[recv.Name]; ok {
			w.addStdlibRoute("", call.Args)
		}
	case *ast.Ident:
		if fun.Name == "HandleFunc" || fun.Name == "Handle" {
			w.addStdlibRoute(prefix, call.Args)
		}
	}
}

func (w *stdlibWalker) addStdlibRoute(prefix string, args []ast.Expr) {
	if len(args) < 2 {
		return
	}
	pattern, ok := stringLiteral(args[0])
	if !ok {
		return
	}

	method := "GET"
	path := pattern
	if parts := strings.SplitN(pattern, " ", 2); len(parts) == 2 && isHTTPMethod(parts[0]) {
		method = strings.ToUpper(parts[0])
		path = parts[1]
	}
	path = joinRoutePath(prefix, path)

	handler, hasHandler := w.ctx.ResolveFuncFromExpr(w.pkg, args[len(args)-1])
	entry := EntryPoint{
		Kind:       protocol.EntryKindHTTP,
		Method:     method,
		Path:       path,
		HasHandler: hasHandler,
	}
	if hasHandler {
		entry.Handler = handler
	}
	w.entries = append(w.entries, entry)
}

func isHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT":
		return true
	default:
		return false
	}
}
