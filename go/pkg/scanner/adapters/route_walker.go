package adapters

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

type routerFrame struct {
	prefix     string
	middleware []HandlerRef
}

type routeWalker struct {
	ctx     *ScanContext
	pkg     *packages.Package
	entries []EntryPoint

	routerVars map[string]routerFrame
}

func (w *routeWalker) walkFile(file *ast.File) {
	w.routerVars = make(map[string]routerFrame)
	ast.Walk(w, file)
}

func (w *routeWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.AssignStmt:
		w.trackRouterAssignments(n)
	case *ast.ExprStmt:
		w.trackRouteCall(n.X)
	case *ast.BlockStmt:
		for _, stmt := range n.List {
			if es, ok := stmt.(*ast.ExprStmt); ok {
				w.trackRouteCall(es.X)
			}
		}
	}

	return w
}

func (w *routeWalker) trackRouterAssignments(stmt *ast.AssignStmt) {
	if len(stmt.Rhs) != 1 {
		return
	}
	call, ok := stmt.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}
	if !isNamedCall(call, "NewRouter") {
		return
	}

	for _, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		w.routerVars[ident.Name] = routerFrame{}
	}
}

func (w *routeWalker) trackRouteCall(expr ast.Expr) {
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
	frame, ok := w.routerVars[recv.Name]
	if !ok {
		return
	}

	switch sel.Sel.Name {
	case "Use", "With":
		w.trackMiddleware(recv.Name, frame, call.Args)
	case "Route", "Mount":
		w.trackNestedRoutes(recv.Name, frame, call.Args)
	default:
		w.trackHTTPMethod(recv.Name, frame, sel.Sel.Name, call.Args)
	}
}

func (w *routeWalker) trackMiddleware(varName string, frame routerFrame, args []ast.Expr) {
	for _, arg := range args {
		if ref, ok := w.ctx.ResolveFuncFromExpr(w.pkg, arg); ok {
			frame.middleware = append(frame.middleware, ref)
		}
	}
	w.routerVars[varName] = frame
}

func (w *routeWalker) trackNestedRoutes(varName string, frame routerFrame, args []ast.Expr) {
	if len(args) < 2 {
		return
	}
	path, ok := stringLiteral(args[0])
	if !ok {
		return
	}
	fnLit, ok := args[1].(*ast.FuncLit)
	if !ok {
		return
	}

	child := routerFrame{
		prefix:     joinRoutePath(frame.prefix, path),
		middleware: append([]HandlerRef{}, frame.middleware...),
	}

	saved := make(map[string]routerFrame)
	if len(fnLit.Type.Params.List) > 0 {
		for _, field := range fnLit.Type.Params.List {
			for _, name := range field.Names {
				saved[name.Name] = w.routerVars[name.Name]
				w.routerVars[name.Name] = child
			}
		}
	}

	for _, stmt := range fnLit.Body.List {
		if es, ok := stmt.(*ast.ExprStmt); ok {
			w.trackRouteCall(es.X)
		}
	}

	for name, prev := range saved {
		w.routerVars[name] = prev
	}
}

func (w *routeWalker) trackHTTPMethod(varName string, frame routerFrame, methodName string, args []ast.Expr) {
	method, ok := httpMethodFromSelector(methodName, args)
	if !ok {
		return
	}
	if len(args) < 2 {
		return
	}
	path, ok := stringLiteral(args[0])
	if !ok {
		return
	}
	handlerExpr := args[len(args)-1]

	handler, hasHandler := w.ctx.ResolveFuncFromExpr(w.pkg, handlerExpr)
	entry := EntryPoint{
		Kind:       protocol.EntryKindHTTP,
		Method:     method,
		Path:       joinRoutePath(frame.prefix, path),
		HasHandler: hasHandler,
		Middleware: append([]HandlerRef{}, frame.middleware...),
	}
	if hasHandler {
		entry.Handler = handler
	}
	w.entries = append(w.entries, entry)
}

func httpMethodFromSelector(name string, args []ast.Expr) (string, bool) {
	switch strings.ToLower(name) {
	case "get":
		return "GET", true
	case "post":
		return "POST", true
	case "put":
		return "PUT", true
	case "patch":
		return "PATCH", true
	case "delete":
		return "DELETE", true
	case "head":
		return "HEAD", true
	case "options":
		return "OPTIONS", true
	case "connect":
		return "CONNECT", true
	case "method":
		if len(args) == 0 {
			return "", false
		}
		method, ok := stringLiteral(args[0])
		if !ok {
			return "", false
		}
		return strings.ToUpper(method), true
	default:
		return "", false
	}
}

func isNamedCall(call *ast.CallExpr, name string) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name == name
	case *ast.SelectorExpr:
		return fun.Sel.Name == name
	default:
		return false
	}
}

func cloneRouterVars(in map[string]routerFrame) map[string]routerFrame {
	out := make(map[string]routerFrame, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
