package adapters

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ScanContext carries module and path helpers for entry discovery.
type ScanContext struct {
	RootModule string
	Fset       *token.FileSet
	PkgByPath  map[string]*packages.Package
}

// NewScanContext builds a scan context from loaded packages.
func NewScanContext(rootModule string, pkgs []*packages.Package) *ScanContext {
	ctx := &ScanContext{
		RootModule: rootModule,
		PkgByPath:  make(map[string]*packages.Package, len(pkgs)),
	}
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		ctx.PkgByPath[pkg.PkgPath] = pkg
		if ctx.Fset == nil && pkg.Fset != nil {
			ctx.Fset = pkg.Fset
		}
	}
	return ctx
}

// RelFilePath converts an absolute file path to a module-relative slash path.
func (c *ScanContext) RelFilePath(absPath string, pkg *packages.Package) string {
	absPath = filepath.Clean(absPath)
	if pkg != nil && pkg.Module != nil && pkg.Module.Dir != "" {
		moduleDir := filepath.Clean(pkg.Module.Dir)
		if rel, err := filepath.Rel(moduleDir, absPath); err == nil && !strings.HasPrefix(rel, "..") {
			return toSlashPath(rel)
		}
	}
	return toSlashPath(absPath)
}

// PackageForFile returns the loaded package that owns absPath, if any.
func (c *ScanContext) PackageForFile(absPath string) *packages.Package {
	absPath = filepath.Clean(absPath)
	for _, pkg := range c.PkgByPath {
		for _, file := range pkg.CompiledGoFiles {
			if filepath.Clean(file) == absPath {
				return pkg
			}
		}
	}
	return nil
}

// ResolveFuncFromExpr resolves an expression to a handler/function reference.
func (c *ScanContext) ResolveFuncFromExpr(pkg *packages.Package, expr ast.Expr) (HandlerRef, bool) {
	if expr == nil || pkg == nil {
		return HandlerRef{}, false
	}

	switch e := expr.(type) {
	case *ast.FuncLit:
		pos := pkg.Fset.Position(e.Pos())
		return HandlerRef{
			File:    c.RelFilePath(pos.Filename, pkg),
			Line:    pos.Line,
			Name:    "<anonymous>",
			Package: pkg.PkgPath,
		}, true
	case *ast.Ident:
		if pkg.TypesInfo != nil {
			if ref, ok := c.funcFromObject(pkg, pkg.TypesInfo.ObjectOf(e)); ok {
				return ref, true
			}
		}
		return c.resolveFuncByName(pkg, e.Name)
	case *ast.SelectorExpr:
		if pkg.TypesInfo != nil {
			if ref, ok := c.funcFromObject(pkg, pkg.TypesInfo.ObjectOf(e.Sel)); ok {
				return ref, true
			}
		}
		return c.resolveSelectorMethod(pkg, e)
	default:
		return HandlerRef{}, false
	}
}

func (c *ScanContext) resolveSelectorMethod(pkg *packages.Package, sel *ast.SelectorExpr) (HandlerRef, bool) {
	if sel.Sel == nil {
		return HandlerRef{}, false
	}
	methodName := sel.Sel.Name

	if ident, ok := sel.X.(*ast.Ident); ok {
		if ownerPkg, typeName := c.inferHandlerTypeFromVar(pkg, ident.Name); ownerPkg != "" && typeName != "" {
			if ref, ok := c.findMethodInPackage(ownerPkg, typeName, methodName); ok {
				return ref, true
			}
		}
		if ownerPkg := c.inferHandlerPackageFromVar(pkg, ident.Name); ownerPkg != "" {
			if ref, ok := c.findFuncNamed(methodName, ownerPkg); ok {
				return ref, true
			}
		}
	}

	return c.findFuncNamed(methodName, pkg.PkgPath)
}

func (c *ScanContext) inferHandlerTypeFromVar(pkg *packages.Package, varName string) (string, string) {
	if pkg == nil || pkg.Syntax == nil || varName == "" {
		return "", ""
	}

	for _, file := range pkg.Syntax {
		var importPath, typeName string
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				return true
			}
			for _, lhs := range assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != varName {
					continue
				}
				importPath, typeName = c.typeFromConstructor(pkg, assign.Rhs[0])
				if importPath != "" && typeName != "" {
					return false
				}
			}
			return true
		})
		if importPath != "" && typeName != "" {
			return importPath, typeName
		}
	}
	return "", ""
}

func (c *ScanContext) typeFromConstructor(pkg *packages.Package, expr ast.Expr) (string, string) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return "", ""
	}
	importPath := c.packageFromConstructor(pkg, expr)
	typeName := typeNameFromConstructor(sel.Sel.Name)
	return importPath, typeName
}

func typeNameFromConstructor(name string) string {
	if !strings.HasPrefix(name, "New") || len(name) <= 3 {
		return ""
	}
	return name[3:]
}

func (c *ScanContext) findMethodInPackage(pkgPath, typeName, methodName string) (HandlerRef, bool) {
	pkg := c.PkgByPath[pkgPath]
	if pkg == nil || pkg.Syntax == nil {
		return HandlerRef{}, false
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != methodName || fn.Recv == nil {
				continue
			}
			if receiverTypeName(fn.Recv) != typeName {
				continue
			}
			pos := c.Fset.Position(fn.Pos())
			return HandlerRef{
				File:    c.RelFilePath(pos.Filename, pkg),
				Line:    pos.Line,
				Name:    fn.Name.Name,
				Package: pkg.PkgPath,
			}, true
		}
	}
	return HandlerRef{}, false
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	typ := recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	if ident, ok := typ.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func (c *ScanContext) inferHandlerPackageFromVar(pkg *packages.Package, varName string) string {
	if pkg == nil || pkg.Syntax == nil || varName == "" {
		return ""
	}

	for _, file := range pkg.Syntax {
		var found string
		ast.Inspect(file, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Rhs) != 1 {
				return true
			}
			for _, lhs := range assign.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != varName {
					continue
				}
				if importPath := c.packageFromConstructor(pkg, assign.Rhs[0]); importPath != "" {
					found = importPath
					return false
				}
			}
			return true
		})
		if found != "" {
			return found
		}
	}
	return ""
}

func (c *ScanContext) packageFromConstructor(pkg *packages.Package, expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	if pkg.TypesInfo != nil {
		if obj := pkg.TypesInfo.ObjectOf(pkgIdent); obj != nil {
			if imp, ok := obj.(*types.PkgName); ok && imp.Imported() != nil {
				return imp.Imported().Path()
			}
		}
	}

	for path, imp := range pkg.Imports {
		importName := imp.Name
		if importName == "" {
			importName = pathBase(path)
		}
		if importName == pkgIdent.Name {
			return path
		}
	}
	return ""
}

func pathBase(importPath string) string {
	if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
		return importPath[idx+1:]
	}
	return importPath
}

func (c *ScanContext) resolveFuncByName(pkg *packages.Package, name string) (HandlerRef, bool) {
	return c.findFuncNamed(name, pkg.PkgPath)
}

func (c *ScanContext) findFuncNamed(name, preferredPkg string) (HandlerRef, bool) {
	if name == "" {
		return HandlerRef{}, false
	}

	searchOrder := make([]*packages.Package, 0, len(c.PkgByPath))
	if preferred, ok := c.PkgByPath[preferredPkg]; ok {
		searchOrder = append(searchOrder, preferred)
	}
	for path, pkg := range c.PkgByPath {
		if path == preferredPkg {
			continue
		}
		searchOrder = append(searchOrder, pkg)
	}

	for _, pkg := range searchOrder {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Name.Name != name {
					continue
				}
				pos := c.Fset.Position(fn.Pos())
				return HandlerRef{
					File:    c.RelFilePath(pos.Filename, pkg),
					Line:    pos.Line,
					Name:    fn.Name.Name,
					Package: pkg.PkgPath,
				}, true
			}
		}
	}
	return HandlerRef{}, false
}

func (c *ScanContext) funcFromObject(pkg *packages.Package, obj types.Object) (HandlerRef, bool) {
	fn, ok := obj.(*types.Func)
	if !ok || fn == nil {
		return HandlerRef{}, false
	}

	pos := c.Fset.Position(fn.Pos())
	owner := pkg
	if fn.Pkg() != nil && fn.Pkg().Path() != "" {
		if loaded, ok := c.PkgByPath[fn.Pkg().Path()]; ok {
			owner = loaded
		}
	}
	if owner == nil {
		owner = pkg
	}

	return HandlerRef{
		File:    c.RelFilePath(pos.Filename, owner),
		Line:    pos.Line,
		Name:    fn.Name(),
		Package: fn.Pkg().Path(),
	}, true
}

// FindFuncByName searches project packages for a function or method by qualified name.
// Accepts "path/to/pkg.Func" or "path/to/pkg.Type.Method".
func (c *ScanContext) FindFuncByName(qualified string) (HandlerRef, bool) {
	qualified = strings.TrimSpace(qualified)
	if qualified == "" {
		return HandlerRef{}, false
	}

	lastDot := strings.LastIndex(qualified, ".")
	if lastDot <= 0 {
		return HandlerRef{}, false
	}
	pkgSuffix := qualified[:lastDot]
	symbol := qualified[lastDot+1:]

	for path, pkg := range c.PkgByPath {
		if !strings.HasSuffix(path, pkgSuffix) && path != pkgSuffix {
			continue
		}
		if pkg.Types != nil {
			if obj := pkg.Types.Scope().Lookup(symbol); obj != nil {
				if ref, ok := c.funcFromObject(pkg, obj); ok {
					return ref, true
				}
			}
		}
		if pkg.TypesInfo != nil {
			for _, name := range pkg.TypesInfo.Defs {
				if name == nil {
					continue
				}
				if fn, ok := name.(*types.Func); ok && fn.Name() == symbol {
					if ref, ok := c.funcFromObject(pkg, fn); ok {
						return ref, true
					}
				}
			}
		}
		if ref, ok := c.findFuncNamed(symbol, path); ok {
			return ref, true
		}
	}
	return HandlerRef{}, false
}

func (c *ScanContext) packagePathFromSelector(pkg *packages.Package, sel *ast.SelectorExpr) string {
	if sel == nil {
		return ""
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}

	if pkg.TypesInfo != nil {
		if obj := pkg.TypesInfo.ObjectOf(pkgIdent); obj != nil {
			if imp, ok := obj.(*types.PkgName); ok && imp.Imported() != nil {
				return imp.Imported().Path()
			}
		}
	}

	if path := c.importPathFromIdent(pkg, pkgIdent.Name); path != "" {
		return path
	}

	for path, imp := range pkg.Imports {
		importName := imp.Name
		if importName == "" {
			importName = pathBase(path)
		}
		if importName == pkgIdent.Name {
			return path
		}
	}
	return ""
}

func (c *ScanContext) importPathFromIdent(pkg *packages.Package, identName string) string {
	if pkg == nil || pkg.Syntax == nil || identName == "" {
		return ""
	}

	for _, file := range pkg.Syntax {
		for _, imp := range file.Imports {
			localName := ""
			if imp.Name != nil {
				localName = imp.Name.Name
			}
			importPath := strings.Trim(imp.Path.Value, `"`)
			if localName == "" {
				localName = pathBase(importPath)
			}
			if localName == identName {
				return importPath
			}
		}
	}
	return ""
}

func (c *ScanContext) findMethodOnType(pkgPath, typeName, methodName string) (HandlerRef, bool) {
	return c.findMethodInPackage(pkgPath, typeName, methodName)
}

func toSlashPath(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
}

func joinRoutePath(prefix, segment string) string {
	if segment == "" {
		return prefix
	}
	if !strings.HasPrefix(segment, "/") {
		segment = "/" + segment
	}
	if prefix == "" || prefix == "/" {
		return segment
	}
	return strings.TrimSuffix(prefix, "/") + segment
}

func stringLiteral(value ast.Expr) (string, bool) {
	lit, ok := value.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(lit.Value, `"`), true
}

func isChiImport(path string) bool {
	return strings.Contains(path, "github.com/go-chi/chi")
}

func isGinImport(path string) bool {
	return path == "github.com/gin-gonic/gin"
}

func isEchoImport(path string) bool {
	return strings.Contains(path, "github.com/labstack/echo")
}

func isGRPCImport(path string) bool {
	return path == "google.golang.org/grpc"
}

func pkgImports(pkgs []*packages.Package, match func(string) bool) bool {
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for path := range pkg.Imports {
			if match(path) {
				return true
			}
		}
	}
	return false
}
