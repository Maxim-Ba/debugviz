package wire

import (
	"go/ast"
	"go/token"
	"strings"
)

type wirePlan struct {
	Configure      bool
	HTTPListen     bool
	ChiMiddleware  bool
	GRPCServer     bool
	CLIRun         bool
	WorkerJobs     []string
	SkipConfigure  bool
	SkipHTTP       bool
	ServiceParam   string
	ChiRouterIdent string
}

func analyzeMain(file *ast.File, cfg Config, ann *AppAnnotation) wirePlan {
	plan := wirePlan{
		Configure:  !hasConfigureCall(file),
		HTTPListen: cfg.httpEnabled(),
		GRPCServer: cfg.grpcEnabled(),
		CLIRun:     cfg.cliEnabled(),
	}
	if file == nil {
		return plan
	}
	if fileHasWireSkip(file) {
		return wirePlan{}
	}

	mainFn := findMainFunc(file)
	if mainFn == nil {
		return plan
	}
	if commentGroupBeforeDecl(file, mainFn, "configure") {
		plan.SkipConfigure = true
		plan.Configure = false
	}
	if commentGroupBeforeDecl(file, mainFn, "http") {
		plan.SkipHTTP = true
		plan.HTTPListen = false
	}

	if plan.Configure && hasConfigureCallInFunc(mainFn) {
		plan.Configure = false
	}
	useListenWrap := len(cfg.Wire.HTTP.RouterFiles) == 0
	if useListenWrap && !plan.SkipHTTP && (plan.HTTPListen || hasListenAndServeInFunc(mainFn)) && !hasHTTPListenWired(mainFn) {
		plan.HTTPListen = true
	} else {
		plan.HTTPListen = false
	}
	if (plan.GRPCServer || hasGRPCNewServerInFunc(mainFn)) && !hasGRPCWired(mainFn) {
		plan.GRPCServer = true
	} else if hasGRPCWired(mainFn) {
		plan.GRPCServer = false
	}
	if (plan.CLIRun || hasCobraExecuteInFunc(mainFn)) && !hasCLIWired(mainFn) {
		plan.CLIRun = true
	} else if hasCLIWired(mainFn) {
		plan.CLIRun = false
	}

	for _, target := range cfg.Wire.Worker.Targets {
		if target.Name == "" {
			continue
		}
		if !hasWorkerJobWired(mainFn, target.Name) {
			plan.WorkerJobs = append(plan.WorkerJobs, target.Name)
		}
	}

	if ann != nil && ann.Name != "" && cfg.ServiceName == "" {
		cfg.ServiceName = ann.Name
	}
	return plan
}

func analyzeRouter(file *ast.File, cfg Config) wirePlan {
	plan := wirePlan{ChiMiddleware: cfg.httpEnabled() || importsChi(file)}
	if file == nil || fileHasWireSkip(file) || !importsChi(file) {
		return wirePlan{}
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if hasChiMiddlewareInFunc(fn) {
			plan.ChiMiddleware = false
			break
		}
		param := chiHandlerServiceParam(fn)
		if param != "" {
			plan.ServiceParam = param
			plan.ChiRouterIdent = chiRouterIdent(fn)
			break
		}
	}
	return plan
}

func findMainFunc(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "main" && fn.Body != nil {
			return fn
		}
	}
	return nil
}

func commentGroupBeforeDecl(file *ast.File, fn *ast.FuncDecl, kind string) bool {
	if file == nil || fn == nil {
		return false
	}
	for _, cg := range file.Comments {
		if cg.End() < fn.Pos() && fn.Pos()-cg.End() < 100 {
			if hasWireSkipComment(cg, kind) {
				return true
			}
		}
	}
	if fn.Doc != nil && hasWireSkipComment(fn.Doc, kind) {
		return true
	}
	return false
}

func hasConfigureCall(file *ast.File) bool {
	fn := findMainFunc(file)
	return fn != nil && hasConfigureCallInFunc(fn)
}

func hasConfigureCallInFunc(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isDebugvizIdent(sel.X) {
			return true
		}
		switch sel.Sel.Name {
		case "ConfigureFromEnv", "ConfigureFromYAML", "Configure":
			found = true
			return false
		}
		return true
	})
	return found
}

func hasHTTPListenWired(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isHTTPMiddlewareCall(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasGRPCWired(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isDebugvizIdent(sel.X) {
			return true
		}
		if sel.Sel.Name == "UnaryServerInterceptor" || sel.Sel.Name == "StreamServerInterceptor" {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasCLIWired(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isDebugvizIdent(sel.X) {
			return true
		}
		if sel.Sel.Name == "RunCLI" {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasWorkerJobWired(fn *ast.FuncDecl, jobName string) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isDebugvizIdent(sel.X) || sel.Sel.Name != "RunJob" {
			return true
		}
		if len(call.Args) >= 2 {
			if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if strings.Trim(lit.Value, `"`) == jobName {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func hasChiMiddlewareInFunc(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isChiMiddlewareCall(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

func chiHandlerServiceParam(fn *ast.FuncDecl) string {
	if !usesChiNewRouter(fn) {
		return ""
	}
	if fn.Type.Params == nil {
		return ""
	}
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			continue
		}
		if isStringType(field.Type) {
			return field.Names[0].Name
		}
	}
	return ""
}

func chiRouterIdent(fn *ast.FuncDecl) string {
	ident := "r"
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || !isChiNewRouterCall(call) {
			return true
		}
		if len(assign.Lhs) > 0 {
			if id, ok := assign.Lhs[0].(*ast.Ident); ok {
				ident = id.Name
			}
		}
		return false
	})
	return ident
}

func usesChiNewRouter(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok && isChiNewRouterCall(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

func importsChi(file *ast.File) bool {
	if file == nil {
		return false
	}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "go-chi/chi") {
			return true
		}
	}
	return false
}

func isDebugvizIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "debugviz"
}

func isStringType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "string"
}

func isHTTPMiddlewareCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && isDebugvizIdent(sel.X) && sel.Sel.Name == "HTTPMiddleware"
}

func isChiMiddlewareCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && isDebugvizIdent(sel.X) && sel.Sel.Name == "ChiMiddleware"
}

func isChiNewRouterCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "NewRouter" {
		return false
	}
	if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "chi" {
		return true
	}
	return false
}

func isListenAndServeCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ListenAndServe" {
		return false
	}
	if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "http" {
		return true
	}
	return false
}

func isGRPCNewServerCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "NewServer" {
		return false
	}
	if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "grpc" {
		return true
	}
	return false
}

func hasListenAndServeInFunc(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok && isListenAndServeCall(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasGRPCNewServerInFunc(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok && isGRPCNewServerCall(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

func hasCobraExecuteInFunc(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if ok && extractExecuteCall(ifStmt) != nil {
			found = true
			return false
		}
		return true
	})
	return found
}

func planHasChanges(plan wirePlan) bool {
	return plan.Configure || plan.HTTPListen || plan.ChiMiddleware || plan.GRPCServer || plan.CLIRun || len(plan.WorkerJobs) > 0
}

func planSummary(plan wirePlan) []string {
	var parts []string
	if plan.Configure {
		parts = append(parts, "configure")
	}
	if plan.HTTPListen {
		parts = append(parts, "http.listen_and_serve")
	}
	if plan.ChiMiddleware {
		parts = append(parts, "http.chi_middleware")
	}
	if plan.GRPCServer {
		parts = append(parts, "grpc.interceptors")
	}
	if plan.CLIRun {
		parts = append(parts, "cli.run")
	}
	for _, job := range plan.WorkerJobs {
		parts = append(parts, "worker."+job)
	}
	return parts
}
