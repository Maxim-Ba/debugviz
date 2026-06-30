package adapters

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

var workerHandlerNames = map[string]struct{}{
	"Consume": {},
	"Handle":  {},
	"Process": {},
}

// Worker discoverer finds background job entry points using best-effort heuristics.
type Worker struct{}

func NewWorker() *Worker { return &Worker{} }

func (w *Worker) Name() string { return "worker" }

func (w *Worker) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	var entries []EntryPoint

	entries = append(entries, discoverWorkerHandlerMethods(ctx, pkgs)...)
	entries = append(entries, discoverRunJobCalls(ctx, pkgs)...)
	if pkgImports(pkgs, isCronImport) {
		entries = append(entries, discoverCronJobs(ctx, pkgs)...)
	}

	return dedupeEntries(entries), nil
}

func discoverWorkerHandlerMethods(ctx *ScanContext, pkgs []*packages.Package) []EntryPoint {
	var entries []EntryPoint

	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Recv == nil {
					continue
				}
				if _, ok := workerHandlerNames[fn.Name.Name]; !ok {
					continue
				}

				typeName := receiverTypeName(fn.Recv)
				if typeName == "" {
					continue
				}

				pos := ctx.Fset.Position(fn.Pos())
				entry := EntryPoint{
					Kind: protocol.EntryKindWorker,
					Job:  typeName + "." + fn.Name.Name,
					Handler: HandlerRef{
						File:    ctx.RelFilePath(pos.Filename, pkg),
						Line:    pos.Line,
						Name:    fn.Name.Name,
						Package: pkg.PkgPath,
					},
					HasHandler: true,
				}
				if queue := inferQueueFromReceiver(ctx, pkg, typeName); queue != "" {
					entry.Queue = queue
				} else if queue := inferQueueFromConstructor(ctx, typeName); queue != "" {
					entry.Queue = queue
				}
				entries = append(entries, entry)
			}
		}
	}

	return entries
}

func discoverRunJobCalls(ctx *ScanContext, pkgs []*packages.Package) []EntryPoint {
	var entries []EntryPoint

	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) < 3 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "RunJob" {
					return true
				}
				if !isDebugvizImport(ctx.packagePathFromSelector(pkg, sel)) {
					return true
				}

				job, ok := stringLiteral(call.Args[1])
				if !ok || job == "" {
					return true
				}

				entry := EntryPoint{
					Kind: protocol.EntryKindWorker,
					Job:  job,
				}
				if ref, ok := ctx.ResolveFuncFromExpr(pkg, call.Args[2]); ok {
					entry.Handler = ref
					entry.HasHandler = true
				}
				entries = append(entries, entry)
				return true
			})
		}
	}

	return entries
}

func discoverCronJobs(ctx *ScanContext, pkgs []*packages.Package) []EntryPoint {
	var entries []EntryPoint

	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) < 2 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel == nil || sel.Sel.Name != "AddFunc" {
					return true
				}

				schedule, ok := stringLiteral(call.Args[0])
				if !ok || schedule == "" {
					return true
				}

				entry := EntryPoint{
					Kind:  protocol.EntryKindWorker,
					Job:   "cron:" + schedule,
					Queue: "cron",
				}
				if ref, ok := ctx.ResolveFuncFromExpr(pkg, call.Args[1]); ok {
					entry.Handler = ref
					entry.HasHandler = true
				}
				entries = append(entries, entry)
				return true
			})
		}
	}

	return entries
}

func inferQueueFromReceiver(ctx *ScanContext, pkg *packages.Package, typeName string) string {
	if pkg == nil || pkg.Syntax == nil || typeName == "" {
		return ""
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name == nil || typeSpec.Name.Name != typeName {
					continue
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						if !strings.EqualFold(name.Name, "queue") {
							continue
						}
						if tag, ok := structTagValue(field.Tag, "queue"); ok {
							return tag
						}
					}
				}
			}
		}
	}

	return inferQueueFromConstructor(ctx, typeName)
}

func inferQueueFromConstructor(ctx *ScanContext, typeName string) string {
	if ctx == nil || typeName == "" {
		return ""
	}

	constructor := "New" + typeName
	for _, pkg := range ctx.PkgByPath {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			var found string
			ast.Inspect(file, func(n ast.Node) bool {
				if found != "" {
					return false
				}
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				if callName(call.Fun) != constructor {
					return true
				}
				if queue, ok := stringLiteral(call.Args[0]); ok {
					found = queue
					return false
				}
				return true
			})
			if found != "" {
				return found
			}
		}
	}
	return ""
}

func callName(expr ast.Expr) string {
	switch fun := expr.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		if fun.Sel != nil {
			return fun.Sel.Name
		}
	}
	return ""
}

func structTagValue(tag *ast.BasicLit, key string) (string, bool) {
	if tag == nil || tag.Kind != token.STRING {
		return "", false
	}
	raw := strings.Trim(tag.Value, "`")
	for _, part := range strings.Fields(raw) {
		if !strings.HasPrefix(part, key+":") {
			continue
		}
		value := strings.TrimPrefix(part, key+":")
		value = strings.Trim(value, `"`)
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func isCronImport(path string) bool {
	return strings.Contains(path, "github.com/robfig/cron")
}

func isDebugvizImport(path string) bool {
	return strings.HasSuffix(path, "/debugviz") ||
		path == "github.com/Maxim-Ba/debugviz/go/lib/debugviz"
}
