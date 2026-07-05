package adapters

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

var registerServerName = regexp.MustCompile(`^Register(.+)Server$`)

// GRPC discoverer finds gRPC entry points from Register*Server calls and generated ServiceDesc.
type GRPC struct{}

func NewGRPC() *GRPC { return &GRPC{} }

func (g *GRPC) Name() string { return "grpc" }

func (g *GRPC) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	if !pkgImports(pkgs, isGRPCImport) {
		return nil, nil
	}

	serviceDescs := collectServiceDescs(ctx, pkgs)
	if len(serviceDescs) == 0 {
		return nil, nil
	}

	var entries []EntryPoint
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				registration, ok := parseRegisterServerCall(ctx, pkg, call)
				if !ok {
					return true
				}

				desc, ok := serviceDescs[registration.serviceDescKey()]
				if !ok {
					return true
				}

				handlerType, hasHandlerType := resolveHandlerType(ctx, pkg, registration.handlerExpr)
				for _, method := range desc.methods {
					entry := EntryPoint{
						Kind:    protocol.EntryKindGRPC,
						Service: desc.serviceName,
						Method:  method,
					}
					if hasHandlerType {
						if ref, ok := ctx.findMethodOnType(handlerType.pkgPath, handlerType.typeName, method); ok {
							entry.Handler = ref
							entry.HasHandler = true
						}
					}
					entries = append(entries, entry)
				}
				return true
			})
		}
	}

	return dedupeEntries(entries), nil
}

type grpcServiceDesc struct {
	serviceName string
	methods     []string
}

type grpcServiceDescKey struct {
	pkgPath     string
	serviceType string
}

type registerServerCall struct {
	pbPkgPath    string
	serviceType  string
	handlerExpr  ast.Expr
}

func (r registerServerCall) serviceDescKey() grpcServiceDescKey {
	return grpcServiceDescKey{
		pkgPath:     r.pbPkgPath,
		serviceType: r.serviceType,
	}
}

type handlerTypeRef struct {
	pkgPath  string
	typeName string
}

func parseRegisterServerCall(ctx *ScanContext, pkg *packages.Package, call *ast.CallExpr) (registerServerCall, bool) {
	if len(call.Args) < 2 {
		return registerServerCall{}, false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return registerServerCall{}, false
	}

	matches := registerServerName.FindStringSubmatch(sel.Sel.Name)
	if len(matches) != 2 || matches[1] == "" {
		return registerServerCall{}, false
	}

	pbPkgPath := ctx.packagePathFromSelector(pkg, sel)
	if pbPkgPath == "" {
		return registerServerCall{}, false
	}

	return registerServerCall{
		pbPkgPath:   pbPkgPath,
		serviceType: matches[1],
		handlerExpr: call.Args[1],
	}, true
}

func collectServiceDescs(ctx *ScanContext, pkgs []*packages.Package) map[grpcServiceDescKey]grpcServiceDesc {
	descs := make(map[grpcServiceDescKey]grpcServiceDesc)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range gen.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok || len(valueSpec.Names) == 0 {
						continue
					}
					name := valueSpec.Names[0].Name
					if !strings.HasSuffix(name, "_ServiceDesc") {
						continue
					}
					serviceType := strings.TrimSuffix(name, "_ServiceDesc")
					if serviceType == "" || len(valueSpec.Values) != 1 {
						continue
					}
					composite, ok := valueSpec.Values[0].(*ast.CompositeLit)
					if !ok {
						continue
					}
					desc, ok := parseServiceDescComposite(composite)
					if !ok || desc.serviceName == "" || len(desc.methods) == 0 {
						continue
					}
					key := grpcServiceDescKey{
						pkgPath:     pkg.PkgPath,
						serviceType: serviceType,
					}
					descs[key] = desc
				}
			}
		}
	}
	return descs
}

func parseServiceDescComposite(composite *ast.CompositeLit) (grpcServiceDesc, bool) {
	var desc grpcServiceDesc
	for _, elt := range composite.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		switch key.Name {
		case "ServiceName":
			if serviceName, ok := stringLiteral(kv.Value); ok {
				desc.serviceName = serviceName
			}
		case "Methods":
			desc.methods = append(desc.methods, parseMethodDescNames(kv.Value)...)
		}
	}
	return desc, desc.serviceName != "" && len(desc.methods) > 0
}

func parseMethodDescNames(expr ast.Expr) []string {
	array, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	var methods []string
	for _, elt := range array.Elts {
		methodLit, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}
		for _, field := range methodLit.Elts {
			kv, ok := field.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "MethodName" {
				continue
			}
			if methodName, ok := stringLiteral(kv.Value); ok && methodName != "" {
				methods = append(methods, methodName)
			}
		}
	}
	return methods
}

func resolveHandlerType(ctx *ScanContext, pkg *packages.Package, expr ast.Expr) (handlerTypeRef, bool) {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op != token.AND {
			return handlerTypeRef{}, false
		}
		return resolveHandlerTypeFromComposite(ctx, pkg, e.X)
	case *ast.CompositeLit:
		return resolveHandlerTypeFromComposite(ctx, pkg, e)
	case *ast.CallExpr:
		return resolveHandlerTypeFromCall(ctx, pkg, e)
	case *ast.Ident:
		return resolveHandlerTypeFromIdent(ctx, pkg, e.Name)
	case *ast.SelectorExpr:
		if pkgPath := ctx.packagePathFromSelector(pkg, e); pkgPath != "" {
			return handlerTypeRef{pkgPath: pkgPath, typeName: e.Sel.Name}, true
		}
	}
	return handlerTypeRef{}, false
}

func resolveHandlerTypeFromComposite(ctx *ScanContext, pkg *packages.Package, expr ast.Expr) (handlerTypeRef, bool) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return handlerTypeRef{}, false
	}

	switch typ := lit.Type.(type) {
	case *ast.Ident:
		return handlerTypeRef{pkgPath: pkg.PkgPath, typeName: typ.Name}, true
	case *ast.SelectorExpr:
		if pkgPath := ctx.packagePathFromSelector(pkg, typ); pkgPath != "" {
			return handlerTypeRef{pkgPath: pkgPath, typeName: typ.Sel.Name}, true
		}
	}
	return handlerTypeRef{}, false
}

func resolveHandlerTypeFromCall(ctx *ScanContext, pkg *packages.Package, call *ast.CallExpr) (handlerTypeRef, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return handlerTypeRef{}, false
	}
	if !strings.HasPrefix(sel.Sel.Name, "New") || len(sel.Sel.Name) <= 3 {
		return handlerTypeRef{}, false
	}
	pkgPath := ctx.packagePathFromSelector(pkg, sel)
	if pkgPath == "" {
		pkgPath = pkg.PkgPath
	}
	return handlerTypeRef{
		pkgPath:  pkgPath,
		typeName: sel.Sel.Name[3:],
	}, true
}

func resolveHandlerTypeFromIdent(ctx *ScanContext, pkg *packages.Package, varName string) (handlerTypeRef, bool) {
	importPath, typeName := ctx.inferHandlerTypeFromVar(pkg, varName)
	if importPath != "" && typeName != "" {
		return handlerTypeRef{pkgPath: importPath, typeName: typeName}, true
	}
	return handlerTypeRef{}, false
}
