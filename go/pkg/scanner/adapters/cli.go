package adapters

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/Maxim-Ba/debugviz/go/pkg/protocol"
)

// CLI discoverer finds CLI entry points from cobra and urfave/cli command trees.
type CLI struct{}

func NewCLI() *CLI { return &CLI{} }

func (c *CLI) Name() string { return "cli" }

func (c *CLI) Discover(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	var entries []EntryPoint

	if pkgImports(pkgs, isCobraImport) {
		found, err := discoverCobraCommands(ctx, pkgs)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}

	if pkgImports(pkgs, isUrfaveCLIImport) {
		found, err := discoverUrfaveCommands(ctx, pkgs)
		if err != nil {
			return nil, err
		}
		entries = append(entries, found...)
	}

	return dedupeEntries(entries), nil
}

type cliCommandDef struct {
	ident    string
	use      string
	run      ast.Expr
	action   ast.Expr
	pkg      *packages.Package
	children []string
}

func discoverCobraCommands(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	commands := make(map[string]*cliCommandDef)
	childOf := make(map[string]string)

	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.AssignStmt:
					trackCobraAssignments(ctx, pkg, node, commands)
				case *ast.CallExpr:
					trackCobraAddCommand(node, commands, childOf)
				}
				return true
			})
		}
	}

	return buildCLIEntries(ctx, commands, childOf), nil
}

func trackCobraAssignments(ctx *ScanContext, pkg *packages.Package, stmt *ast.AssignStmt, commands map[string]*cliCommandDef) {
	if len(stmt.Rhs) != 1 {
		return
	}
	lit, ok := stmt.Rhs[0].(*ast.UnaryExpr)
	if !ok || lit.Op != token.AND {
		return
	}
	composite, ok := lit.X.(*ast.CompositeLit)
	if !ok || !isCobraCommandType(ctx, pkg, composite.Type) {
		return
	}

	use, run, _ := parseCobraCommandFields(composite)
	for _, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		commands[ident.Name] = &cliCommandDef{
			ident: ident.Name,
			use:   use,
			run:   run,
			pkg:   pkg,
		}
	}
}

func trackCobraAddCommand(call *ast.CallExpr, commands map[string]*cliCommandDef, childOf map[string]string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "AddCommand" {
		return
	}
	parent, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}
	if _, ok := commands[parent.Name]; !ok {
		return
	}

	for i, arg := range call.Args {
		childIdent, def := parseCobraCommandArg(arg, commands, parent.Name, i)
		if childIdent == "" || def == nil {
			continue
		}
		commands[childIdent] = def
		commands[parent.Name].children = appendUniqueString(commands[parent.Name].children, childIdent)
		childOf[childIdent] = parent.Name
	}
}

func parseCobraCommandArg(expr ast.Expr, commands map[string]*cliCommandDef, parentName string, index int) (string, *cliCommandDef) {
	if ident, ok := expr.(*ast.Ident); ok {
		if def, ok := commands[ident.Name]; ok {
			return ident.Name, def
		}
		return ident.Name, nil
	}

	lit, ok := expr.(*ast.UnaryExpr)
	if !ok || lit.Op != token.AND {
		return "", nil
	}
	composite, ok := lit.X.(*ast.CompositeLit)
	if !ok {
		return "", nil
	}
	use, run, ok := parseCobraCommandFields(composite)
	if !ok || use == "" {
		return "", nil
	}

	ident := fmt.Sprintf("%s:inline:%d", parentName, index)
	return ident, &cliCommandDef{
		ident: ident,
		use:   use,
		run:   run,
	}
}

func parseCobraCommandFields(composite *ast.CompositeLit) (use string, run ast.Expr, ok bool) {
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
		case "Use":
			use, _ = stringLiteral(kv.Value)
		case "Run":
			run = kv.Value
		}
	}
	return use, run, use != ""
}

func isCobraCommandType(ctx *ScanContext, pkg *packages.Package, typ ast.Expr) bool {
	sel, ok := typ.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Command" {
		return false
	}
	return ctx.packagePathFromSelector(pkg, sel) == "github.com/spf13/cobra"
}

func discoverUrfaveCommands(ctx *ScanContext, pkgs []*packages.Package) ([]EntryPoint, error) {
	commands := make(map[string]*cliCommandDef)
	childOf := make(map[string]string)

	for _, pkg := range pkgs {
		if pkg == nil || pkg.Syntax == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.AssignStmt:
					trackUrfaveAssignments(ctx, pkg, node, commands, childOf)
				case *ast.CompositeLit:
					if isUrfaveAppType(ctx, pkg, node.Type) {
						trackUrfaveAppCommands(ctx, pkg, node, "app", commands, childOf)
					}
				}
				return true
			})
		}
	}

	return buildCLIEntries(ctx, commands, childOf), nil
}

func trackUrfaveAssignments(ctx *ScanContext, pkg *packages.Package, stmt *ast.AssignStmt, commands map[string]*cliCommandDef, childOf map[string]string) {
	if len(stmt.Rhs) != 1 {
		return
	}

	switch rhs := stmt.Rhs[0].(type) {
	case *ast.UnaryExpr:
		if rhs.Op != token.AND {
			return
		}
		composite, ok := rhs.X.(*ast.CompositeLit)
		if !ok {
			return
		}
		if isUrfaveAppType(ctx, pkg, composite.Type) {
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					trackUrfaveAppCommands(ctx, pkg, composite, ident.Name, commands, childOf)
				}
			}
			return
		}
		if isUrfaveCommandType(ctx, pkg, composite.Type) {
			use, action, ok := parseUrfaveCommandFields(composite)
			if !ok {
				return
			}
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					commands[ident.Name] = &cliCommandDef{
						ident:  ident.Name,
						use:    use,
						action: action,
						pkg:    pkg,
					}
					trackUrfaveNestedCommands(ctx, pkg, composite, ident.Name, commands, childOf)
				}
			}
		}
	case *ast.CompositeLit:
		if !isUrfaveCommandType(ctx, pkg, rhs.Type) {
			return
		}
		use, action, ok := parseUrfaveCommandFields(rhs)
		if !ok {
			return
		}
		for _, lhs := range stmt.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				commands[ident.Name] = &cliCommandDef{
					ident:  ident.Name,
					use:    use,
					action: action,
					pkg:    pkg,
				}
				trackUrfaveNestedCommands(ctx, pkg, rhs, ident.Name, commands, childOf)
			}
		}
	}
}

func trackUrfaveAppCommands(ctx *ScanContext, pkg *packages.Package, app *ast.CompositeLit, appIdent string, commands map[string]*cliCommandDef, childOf map[string]string) {
	for _, elt := range app.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Commands" {
			continue
		}
		trackUrfaveCommandSlice(ctx, pkg, kv.Value, appIdent, commands, childOf)
	}
}

func trackUrfaveNestedCommands(ctx *ScanContext, pkg *packages.Package, composite *ast.CompositeLit, parentIdent string, commands map[string]*cliCommandDef, childOf map[string]string) {
	for _, elt := range composite.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Subcommands" {
			continue
		}
		trackUrfaveCommandSlice(ctx, pkg, kv.Value, parentIdent, commands, childOf)
	}
}

func trackUrfaveCommandSlice(ctx *ScanContext, pkg *packages.Package, expr ast.Expr, parentIdent string, commands map[string]*cliCommandDef, childOf map[string]string) {
	array, ok := expr.(*ast.CompositeLit)
	if !ok {
		return
	}

	for i, elt := range array.Elts {
		lit := elt
		if unary, ok := elt.(*ast.UnaryExpr); ok && unary.Op == token.AND {
			lit = unary.X
		}
		composite, ok := lit.(*ast.CompositeLit)
		if !ok {
			continue
		}
		use, action, ok := parseUrfaveCommandFields(composite)
		if !ok {
			continue
		}

		ident := fmt.Sprintf("%s:urfave:%d", parentIdent, i)
		commands[ident] = &cliCommandDef{
			ident:  ident,
			use:    use,
			action: action,
			pkg:    pkg,
		}
		if parentIdent != "" {
			if parent, ok := commands[parentIdent]; ok {
				parent.children = appendUniqueString(parent.children, ident)
			}
			childOf[ident] = parentIdent
		}
		trackUrfaveNestedCommands(ctx, pkg, composite, ident, commands, childOf)
	}
}

func parseUrfaveCommandFields(composite *ast.CompositeLit) (name string, action ast.Expr, ok bool) {
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
		case "Name":
			name, _ = stringLiteral(kv.Value)
		case "Action":
			action = kv.Value
		}
	}
	return name, action, name != ""
}

func isUrfaveAppType(ctx *ScanContext, pkg *packages.Package, typ ast.Expr) bool {
	sel, ok := typ.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "App" {
		return false
	}
	path := ctx.packagePathFromSelector(pkg, sel)
	return isUrfaveCLIImport(path)
}

func isUrfaveCommandType(ctx *ScanContext, pkg *packages.Package, typ ast.Expr) bool {
	sel, ok := typ.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Command" {
		return false
	}
	path := ctx.packagePathFromSelector(pkg, sel)
	return isUrfaveCLIImport(path)
}

func buildCLIEntries(ctx *ScanContext, commands map[string]*cliCommandDef, childOf map[string]string) []EntryPoint {
	if len(commands) == 0 {
		return nil
	}

	var entries []EntryPoint
	for ident, cmd := range commands {
		handlerExpr := cmd.run
		if handlerExpr == nil {
			handlerExpr = cmd.action
		}
		if handlerExpr == nil {
			continue
		}

		path := commandPath(commands, childOf, ident)
		if path == "" {
			continue
		}

		entry := EntryPoint{
			Kind:    protocol.EntryKindCLI,
			Command: path,
		}
		if ref, ok := ctx.ResolveFuncFromExpr(cmd.pkg, handlerExpr); ok {
			entry.Handler = ref
			entry.HasHandler = true
		}
		entries = append(entries, entry)
	}
	return entries
}

func commandPath(commands map[string]*cliCommandDef, childOf map[string]string, ident string) string {
	var parts []string
	var ids []string

	current := ident
	for current != "" {
		cmd, ok := commands[current]
		if !ok {
			break
		}
		if token := commandUseToken(cmd.use); token != "" {
			parts = append([]string{token}, parts...)
			ids = append([]string{current}, ids...)
		}
		current = childOf[current]
	}

	if len(parts) == 0 {
		return ""
	}
	if len(parts) > 1 {
		rootCmd := commands[ids[0]]
		if rootCmd != nil && rootCmd.run == nil && rootCmd.action == nil && childOf[ids[0]] == "" {
			return strings.Join(parts[1:], " ")
		}
	}
	return strings.Join(parts, " ")
}

func commandUseToken(use string) string {
	use = strings.TrimSpace(use)
	if use == "" {
		return ""
	}
	if idx := strings.Index(use, " "); idx >= 0 {
		return use[:idx]
	}
	return use
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func isCobraImport(path string) bool {
	return path == "github.com/spf13/cobra"
}

func isUrfaveCLIImport(path string) bool {
	return path == "github.com/urfave/cli/v2" || path == "github.com/urfave/cli"
}
