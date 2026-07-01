package instrument

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
)

func typeCheckFile(fset *token.FileSet, file *ast.File, pkgName string) (*types.Package, *types.Info) {
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	cfg := &types.Config{
		Importer: importer.Default(),
	}
	pkg, err := cfg.Check(pkgName, fset, []*ast.File{file}, info)
	if err != nil {
		return types.NewPackage(pkgName, pkgName), info
	}
	return pkg, info
}
