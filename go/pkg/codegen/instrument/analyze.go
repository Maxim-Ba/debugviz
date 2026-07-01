package instrument

import (
	"go/ast"
	"go/types"
	"strings"
)

const instrumentSkipDirective = "debugviz:instrument skip"

// FuncCandidate describes a function eligible for span injection.
type FuncCandidate struct {
	Decl       *ast.FuncDecl
	SpanName   string
	HasContext bool
	SkipReason string
}

func analyzeFile(file *ast.File, pkg *types.Package, info *types.Info, cfg Config) []FuncCandidate {
	if file == nil || pkg == nil {
		return nil
	}

	var out []FuncCandidate
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		candidate, ok := analyzeFunc(fn, pkg, info, cfg)
		if !ok {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func analyzeFunc(fn *ast.FuncDecl, pkg *types.Package, info *types.Info, cfg Config) (FuncCandidate, bool) {
	if fn == nil || fn.Body == nil {
		return FuncCandidate{}, false
	}
	if fn.Name == nil {
		return FuncCandidate{}, false
	}
	name := fn.Name.Name
	if name == "init" || name == "main" {
		return FuncCandidate{}, false
	}
	if funcHasSkipDirective(fn) {
		return FuncCandidate{}, false
	}
	if isAlreadyInstrumented(fn.Body) {
		return FuncCandidate{}, false
	}

	hasCtx := firstParamIsContext(fn, info)
	if !hasCtx && !cfg.AllowNoContext {
		return FuncCandidate{}, false
	}

	return FuncCandidate{
		Decl:       fn,
		SpanName:   spanName(pkg, fn),
		HasContext: hasCtx,
	}, true
}

func funcHasSkipDirective(fn *ast.FuncDecl) bool {
	if hasSkipDirective(fn.Doc) {
		return true
	}
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CommentGroup:
			if hasSkipDirective(node) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func hasSkipDirective(comment *ast.CommentGroup) bool {
	if comment == nil {
		return false
	}
	for _, c := range comment.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		if text == instrumentSkipDirective {
			return true
		}
	}
	return false
}

func isAlreadyInstrumented(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) == 0 {
		return false
	}

	for i := 0; i < len(body.List) && i < 3; i++ {
		switch stmt := body.List[i].(type) {
		case *ast.AssignStmt:
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == "__dv_end" {
					return true
				}
			}
			if call, ok := stmt.Rhs[0].(*ast.CallExpr); ok && isStartSpanCall(call) {
				return true
			}
		case *ast.DeferStmt:
			call := stmt.Call
			if call != nil {
				if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "__dv_end" {
					return true
				}
			}
		}
	}
	return false
}

func isStartSpanCall(call *ast.CallExpr) bool {
	if call == nil {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "debugviz" {
		return false
	}
	return sel.Sel != nil && sel.Sel.Name == "StartSpan"
}

func firstParamIsContext(fn *ast.FuncDecl, info *types.Info) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	first := fn.Type.Params.List[0]
	if len(first.Names) == 0 {
		return false
	}
	tv := info.TypeOf(first.Type)
	if tv == nil {
		return false
	}
	named, ok := tv.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

func spanName(pkg *types.Package, fn *ast.FuncDecl) string {
	pkgName := pkg.Name()
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recvType := typeName(fn.Recv.List[0].Type)
		if recvType != "" {
			return pkgName + "." + recvType + "." + fn.Name.Name
		}
	}
	return pkgName + "." + fn.Name.Name
}

func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeName(t.X)
	case *ast.IndexExpr:
		return typeName(t.X)
	case *ast.IndexListExpr:
		return typeName(t.X)
	default:
		return ""
	}
}

func shouldInstrumentFile(relPath string, cfg Config, includeTests bool) bool {
	if !includeTests && strings.HasSuffix(relPath, "_test.go") {
		return false
	}
	if pathMatchesAny(relPath, cfg.Exclude) {
		return false
	}

	includePatterns := cfg.Include
	if len(includePatterns) == 0 && len(cfg.EntryPackages) > 0 {
		includePatterns = cfg.EntryPackages
	}
	if len(includePatterns) == 0 {
		return true
	}
	if pathMatchesAny(relPath, includePatterns) {
		return true
	}
	return pathMatchesAny(relPath, cfg.EntryPackages)
}
