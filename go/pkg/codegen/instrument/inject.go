package instrument

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

func injectCandidates(file *ast.File, fset *token.FileSet, candidates []FuncCandidate, cfg Config) ([]byte, error) {
	if file == nil || len(candidates) == 0 {
		return nil, nil
	}

	for _, candidate := range candidates {
		injectFunc(candidate)
	}

	if err := ensureImports(fset, file, cfg, candidates); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("format file: %w", err)
	}
	return buf.Bytes(), nil
}

func injectFunc(candidate FuncCandidate) {
	fn := candidate.Decl
	if fn == nil || fn.Body == nil {
		return
	}

	spanLit := strconv.Quote(candidate.SpanName)
	var assign *ast.AssignStmt
	if candidate.HasContext {
		ctxName := contextParamName(fn)
		assign = &ast.AssignStmt{
			Lhs: []ast.Expr{
				ast.NewIdent(ctxName),
				ast.NewIdent("__dv_end"),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				startSpanCall(ast.NewIdent(ctxName), spanLit),
			},
		}
	} else {
		assign = &ast.AssignStmt{
			Lhs: []ast.Expr{
				ast.NewIdent("_"),
				ast.NewIdent("__dv_end"),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				startSpanCall(&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("context"),
						Sel: ast.NewIdent("Background"),
					},
				}, spanLit),
			},
		}
	}

	deferStmt := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: ast.NewIdent("__dv_end"),
		},
	}

	fn.Body.List = append([]ast.Stmt{assign, deferStmt}, fn.Body.List...)
}

// contextParamName returns the first parameter name for context.Context, renaming "_" to "ctx".
func contextParamName(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return "ctx"
	}
	first := fn.Type.Params.List[0]
	if len(first.Names) == 0 {
		first.Names = []*ast.Ident{ast.NewIdent("ctx")}
		return "ctx"
	}
	if first.Names[0].Name == "_" {
		first.Names[0].Name = "ctx"
		return "ctx"
	}
	return first.Names[0].Name
}

func startSpanCall(ctxExpr ast.Expr, spanName string) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("debugviz"),
			Sel: ast.NewIdent("StartSpan"),
		},
		Args: []ast.Expr{
			ctxExpr,
			&ast.BasicLit{Kind: token.STRING, Value: spanName},
		},
	}
}

func ensureImports(fset *token.FileSet, file *ast.File, cfg Config, candidates []FuncCandidate) error {
	needsDebugviz := len(candidates) > 0
	needsContext := false
	for _, c := range candidates {
		if !c.HasContext {
			needsContext = true
			break
		}
	}
	if !needsDebugviz && !needsContext {
		return nil
	}

	if needsDebugviz {
		_ = astutil.AddImport(fset, file, cfg.importPath())
	}
	if needsContext {
		_ = astutil.AddImport(fset, file, "context")
	}
	return nil
}

func parseSource(filename string, src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", filename, err)
	}
	return fset, file, nil
}
