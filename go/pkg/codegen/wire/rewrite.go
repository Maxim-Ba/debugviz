package wire

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

func rewriteFile(src []byte, filename string, cfg Config, ann *AppAnnotation, isMain bool) ([]byte, wirePlan, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, wirePlan{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	var plan wirePlan
	if isMain {
		plan = analyzeMain(file, cfg, ann)
	} else {
		plan = analyzeRouter(file, cfg)
	}
	if !planHasChanges(plan) {
		return src, plan, nil
	}

	if err := applyPlan(file, fset, cfg, ann, plan, isMain); err != nil {
		return nil, plan, err
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, plan, fmt.Errorf("format %s: %w", filename, err)
	}
	return buf.Bytes(), plan, nil
}

func applyPlan(file *ast.File, fset *token.FileSet, cfg Config, ann *AppAnnotation, plan wirePlan, isMain bool) error {
	if isMain {
		mainFn := findMainFunc(file)
		if mainFn == nil {
			return fmt.Errorf("main function not found")
		}
		if plan.Configure {
			injectConfigure(mainFn)
		}
		if plan.HTTPListen {
			if err := wrapListenAndServeCalls(mainFn, cfg, ann); err != nil {
				return err
			}
		}
		if plan.GRPCServer {
			injectGRPCInterceptors(mainFn)
		}
		if plan.CLIRun {
			if err := injectCLIRun(mainFn, cfg, ann); err != nil {
				return err
			}
		}
		for _, jobName := range plan.WorkerJobs {
			if err := injectWorkerJob(mainFn, jobName); err != nil {
				return err
			}
		}
	} else if plan.ChiMiddleware {
		if err := injectChiMiddleware(file, cfg, ann); err != nil {
			return err
		}
	} else if plan.GinMiddleware {
		if err := injectGinMiddleware(file, cfg, ann); err != nil {
			return err
		}
	} else if plan.EchoMiddleware {
		if err := injectEchoMiddleware(file, cfg, ann); err != nil {
			return err
		}
	}
	return ensureImports(fset, file, cfg, plan)
}

func injectConfigure(mainFn *ast.FuncDecl) {
	stmt := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("err")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("debugviz"),
						Sel: ast.NewIdent("ConfigureFromEnv"),
					},
				},
			},
		},
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("log"),
							Sel: ast.NewIdent("Fatalf"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: `"debugviz: %v"`},
							ast.NewIdent("err"),
						},
					},
				},
			},
		},
	}
	mainFn.Body.List = append([]ast.Stmt{stmt}, mainFn.Body.List...)
}

func wrapListenAndServeCalls(mainFn *ast.FuncDecl, cfg Config, ann *AppAnnotation) error {
	ast.Inspect(mainFn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isListenAndServeCall(call) || len(call.Args) < 2 {
			return true
		}
		if inner, ok := call.Args[1].(*ast.CallExpr); ok && isHTTPMiddlewareCall(inner) {
			return true
		}
		call.Args[1] = httpMiddlewareCall(call.Args[1], cfg, ann, "")
		return true
	})
	return nil
}

func injectChiMiddleware(file *ast.File, cfg Config, ann *AppAnnotation) error {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || hasChiMiddlewareInFunc(fn) {
			continue
		}
		if !usesChiNewRouter(fn) {
			continue
		}
		routerIdent := chiRouterIdent(fn)
		serviceParam := chiHandlerServiceParam(fn)
		stmt := chiMiddlewareUseStmt(routerIdent, cfg, ann, serviceParam)
		insertAfterRouterCreate(fn, routerIdent, stmt, isChiNewRouterCall)
		return nil
	}
	return fmt.Errorf("chi router function not found")
}

func injectGinMiddleware(file *ast.File, cfg Config, ann *AppAnnotation) error {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || hasGinMiddlewareInFunc(fn) {
			continue
		}
		if !usesGinNew(fn) {
			continue
		}
		routerIdent := ginRouterIdent(fn)
		serviceParam := ginHandlerServiceParam(fn)
		stmt := ginMiddlewareUseStmt(routerIdent, cfg, ann, serviceParam)
		insertAfterRouterCreate(fn, routerIdent, stmt, isGinNewCall)
		return nil
	}
	return fmt.Errorf("gin router function not found")
}

func injectEchoMiddleware(file *ast.File, cfg Config, ann *AppAnnotation) error {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || hasEchoMiddlewareInFunc(fn) {
			continue
		}
		if !usesEchoNew(fn) {
			continue
		}
		routerIdent := echoRouterIdent(fn)
		serviceParam := echoHandlerServiceParam(fn)
		stmt := echoMiddlewareUseStmt(routerIdent, cfg, ann, serviceParam)
		insertAfterRouterCreate(fn, routerIdent, stmt, isEchoNewCall)
		return nil
	}
	return fmt.Errorf("echo router function not found")
}

func chiMiddlewareUseStmt(routerIdent string, cfg Config, ann *AppAnnotation, serviceParam string) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(routerIdent),
				Sel: ast.NewIdent("Use"),
			},
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("debugviz"),
						Sel: ast.NewIdent("ChiMiddleware"),
					},
					Args: []ast.Expr{
						httpMiddlewareConfig(cfg, ann, serviceParam),
					},
				},
			},
		},
	}
}

func ginMiddlewareUseStmt(routerIdent string, cfg Config, ann *AppAnnotation, serviceParam string) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(routerIdent),
				Sel: ast.NewIdent("Use"),
			},
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("debugviz"),
						Sel: ast.NewIdent("GinMiddleware"),
					},
					Args: []ast.Expr{
						httpMiddlewareConfig(cfg, ann, serviceParam),
					},
				},
			},
		},
	}
}

func echoMiddlewareUseStmt(routerIdent string, cfg Config, ann *AppAnnotation, serviceParam string) ast.Stmt {
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(routerIdent),
				Sel: ast.NewIdent("Use"),
			},
			Args: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("debugviz"),
						Sel: ast.NewIdent("EchoMiddleware"),
					},
					Args: []ast.Expr{
						httpMiddlewareConfig(cfg, ann, serviceParam),
					},
				},
			},
		},
	}
}

func insertAfterRouterCreate(fn *ast.FuncDecl, routerIdent string, stmt ast.Stmt, isCreateCall func(*ast.CallExpr) bool) {
	lastUseIdx := -1
	for i, s := range fn.Body.List {
		exprStmt, ok := s.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != routerIdent || sel.Sel.Name != "Use" {
			continue
		}
		lastUseIdx = i
	}
	if lastUseIdx >= 0 {
		fn.Body.List = append(fn.Body.List[:lastUseIdx+1], append([]ast.Stmt{stmt}, fn.Body.List[lastUseIdx+1:]...)...)
		return
	}
	for i, s := range fn.Body.List {
		assign, ok := s.(*ast.AssignStmt)
		if !ok {
			continue
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if ok && isCreateCall(call) {
			fn.Body.List = append(fn.Body.List[:i+1], append([]ast.Stmt{stmt}, fn.Body.List[i+1:]...)...)
			return
		}
	}
	fn.Body.List = append([]ast.Stmt{stmt}, fn.Body.List...)
}

func injectGRPCInterceptors(mainFn *ast.FuncDecl) {
	ast.Inspect(mainFn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isGRPCNewServerCall(call) {
			return true
		}
		if grpcCallHasDebugvizInterceptors(call) {
			return true
		}
		opts := []ast.Expr{
			grpcChainCall("ChainUnaryInterceptor", "UnaryServerInterceptor"),
			grpcChainCall("ChainStreamInterceptor", "StreamServerInterceptor"),
		}
		call.Args = append(opts, call.Args...)
		return true
	})
}

func grpcCallHasDebugvizInterceptors(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		inner, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := inner.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "ChainUnaryInterceptor" {
			continue
		}
		if len(inner.Args) == 0 {
			continue
		}
		probe, ok := inner.Args[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		probeSel, ok := probe.Fun.(*ast.SelectorExpr)
		if ok && isDebugvizIdent(probeSel.X) {
			return true
		}
	}
	return false
}

func grpcChainCall(chainMethod, interceptorMethod string) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("grpc"),
			Sel: ast.NewIdent(chainMethod),
		},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("debugviz"),
					Sel: ast.NewIdent(interceptorMethod),
				},
			},
		},
	}
}

func injectCLIRun(mainFn *ast.FuncDecl, cfg Config, ann *AppAnnotation) error {
	appName := cfg.ServiceName
	if ann != nil && ann.Name != "" {
		appName = ann.Name
	}
	if appName == "" {
		appName = "app"
	}

	injectCobraPreRun(mainFn)

	for i, stmt := range mainFn.Body.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		call := extractExecuteCall(ifStmt)
		if call == nil {
			continue
		}
		executeExpr := callExprToExpr(call)
		mainFn.Body.List[i] = buildRunCLIStmt(appName, executeExpr)
		return nil
	}
	return fmt.Errorf("cobra Execute call not found in main")
}

func injectCobraPreRun(mainFn *ast.FuncDecl) {
	for _, stmt := range mainFn.Body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 {
			continue
		}
		sel, ok := assign.Lhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "PersistentPreRun" {
			continue
		}
		if isDebugvizSelector(assign.Rhs[0], "CLICommandPreRun") {
			return
		}
		assign.Rhs = []ast.Expr{
			&ast.SelectorExpr{
				X:   ast.NewIdent("debugviz"),
				Sel: ast.NewIdent("CLICommandPreRun"),
			},
		}
		return
	}
}

func extractExecuteCall(ifStmt *ast.IfStmt) *ast.CallExpr {
	assign, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok {
		return nil
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	switch sel.Sel.Name {
	case "Execute", "ExecuteContext":
		return call
	}
	return nil
}

func buildRunCLIStmt(appName string, executeExpr ast.Expr) *ast.IfStmt {
	return &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("err")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("debugviz"),
						Sel: ast.NewIdent("RunCLI"),
					},
					Args: []ast.Expr{
						&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(appName)},
						&ast.FuncLit{
							Type: &ast.FuncType{
								Params: &ast.FieldList{
									List: []*ast.Field{
										{
											Names: []*ast.Ident{ast.NewIdent("ctx")},
											Type: &ast.SelectorExpr{
												X:   ast.NewIdent("context"),
												Sel: ast.NewIdent("Context"),
											},
										},
									},
								},
								Results: &ast.FieldList{
									List: []*ast.Field{{Type: ast.NewIdent("error")}},
								},
							},
							Body: &ast.BlockStmt{
								List: []ast.Stmt{
									&ast.ReturnStmt{Results: []ast.Expr{executeExpr}},
								},
							},
						},
					},
				},
			},
		},
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent("err"),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("fmt"),
							Sel: ast.NewIdent("Fprintln"),
						},
						Args: []ast.Expr{
							&ast.SelectorExpr{X: ast.NewIdent("os"), Sel: ast.NewIdent("Stderr")},
							ast.NewIdent("err"),
						},
					},
				},
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{X: ast.NewIdent("os"), Sel: ast.NewIdent("Exit")},
						Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}},
					},
				},
			},
		},
	}
}

func callExprToExpr(call *ast.CallExpr) ast.Expr {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return call
	}
	if sel.Sel.Name == "ExecuteContext" {
		if len(call.Args) == 0 {
			call.Args = []ast.Expr{ast.NewIdent("ctx")}
		} else {
			call.Args[0] = ast.NewIdent("ctx")
		}
		return call
	}
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: sel.X, Sel: ast.NewIdent("ExecuteContext")},
		Args: []ast.Expr{ast.NewIdent("ctx")},
	}
}

func injectWorkerJob(mainFn *ast.FuncDecl, jobName string) error {
	wrapped := false
	ast.Inspect(mainFn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Process" {
			return true
		}
		ctxArg := workerContextArg(call)
		assign.Rhs[0] = &ast.CallExpr{
			Fun: &ast.SelectorExpr{X: ast.NewIdent("debugviz"), Sel: ast.NewIdent("RunJob")},
			Args: []ast.Expr{
				ctxArg,
				&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(jobName)},
				&ast.FuncLit{
					Type: &ast.FuncType{
						Params: &ast.FieldList{
							List: []*ast.Field{{
								Names: []*ast.Ident{ast.NewIdent("ctx")},
								Type:  &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Context")},
							}},
						},
						Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("error")}}},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.ReturnStmt{
								Results: []ast.Expr{
									&ast.CallExpr{Fun: call.Fun, Args: []ast.Expr{ast.NewIdent("ctx")}},
								},
							},
						},
					},
				},
			},
		}
		wrapped = true
		return false
	})
	if !wrapped {
		return fmt.Errorf("worker Process call not found for job %q", jobName)
	}
	return nil
}

func workerContextArg(call *ast.CallExpr) ast.Expr {
	if len(call.Args) == 0 {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{X: ast.NewIdent("context"), Sel: ast.NewIdent("Background")},
		}
	}
	return call.Args[0]
}

func httpMiddlewareCall(handler ast.Expr, cfg Config, ann *AppAnnotation, serviceParam string) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{X: ast.NewIdent("debugviz"), Sel: ast.NewIdent("HTTPMiddleware")},
		Args: []ast.Expr{handler, httpMiddlewareConfig(cfg, ann, serviceParam)},
	}
}

func httpMiddlewareConfig(cfg Config, ann *AppAnnotation, serviceParam string) *ast.CompositeLit {
	return &ast.CompositeLit{
		Type: &ast.SelectorExpr{X: ast.NewIdent("debugviz"), Sel: ast.NewIdent("HTTPMiddlewareConfig")},
		Elts: []ast.Expr{
			&ast.KeyValueExpr{Key: ast.NewIdent("ServiceName"), Value: serviceNameExpr(cfg, ann, serviceParam)},
		},
	}
}

func isDebugvizSelector(expr ast.Expr, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isDebugvizIdent(sel.X) && sel.Sel.Name == name
}

func ensureImports(fset *token.FileSet, file *ast.File, cfg Config, plan wirePlan) error {
	if planHasChanges(plan) {
		_ = astutil.AddImport(fset, file, cfg.importPath())
	}
	if plan.Configure || plan.CLIRun {
		_ = astutil.AddImport(fset, file, "log")
	}
	if plan.CLIRun {
		_ = astutil.AddImport(fset, file, "context")
		_ = astutil.AddImport(fset, file, "fmt")
		_ = astutil.AddImport(fset, file, "os")
	}
	if plan.HTTPListen {
		_ = astutil.AddImport(fset, file, "net/http")
	}
	if plan.GRPCServer {
		_ = astutil.AddImport(fset, file, "google.golang.org/grpc")
	}
	if len(plan.WorkerJobs) > 0 {
		_ = astutil.AddImport(fset, file, "context")
	}
	return nil
}

// WireFile wires a single Go source file. Used by golden tests.
func WireFile(src []byte, filename string, cfg Config, isMain bool) ([]byte, wirePlan, error) {
	ann, err := parseAppAnnotationFromSource(src, filename)
	if err != nil {
		return nil, wirePlan{}, err
	}
	if ann != nil && ann.Name != "" && cfg.ServiceName == "" {
		cfg.ServiceName = ann.Name
	}
	return rewriteFile(src, filename, cfg, ann, isMain)
}

func parseAppAnnotationFromSource(src []byte, filename string) (*AppAnnotation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return parseAppAnnotation(file)
}
