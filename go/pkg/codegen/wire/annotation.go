package wire

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// AppAnnotation holds //debugviz:app metadata from a source file.
type AppAnnotation struct {
	Name   string
	Server string
	Raw    string
}

func parseAppAnnotation(file *ast.File) (*AppAnnotation, error) {
	if file == nil {
		return nil, nil
	}
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if !strings.HasPrefix(text, "debugviz:app") {
				continue
			}
			ann, err := parseAppDirective(text)
			if err != nil {
				return nil, fmt.Errorf("debugviz:app: %w", err)
			}
			return ann, nil
		}
	}
	return nil, nil
}

func parseAppDirective(text string) (*AppAnnotation, error) {
	rest := strings.TrimSpace(strings.TrimPrefix(text, "debugviz:app"))
	if rest == text {
		return nil, fmt.Errorf("invalid debugviz:app annotation %q", text)
	}
	ann := &AppAnnotation{Raw: text}
	if rest == "" {
		return ann, nil
	}
	for _, part := range strings.Fields(rest) {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid debugviz:app token %q (want key=value)", part)
		}
		switch key {
		case "name":
			ann.Name = value
		case "server":
			ann.Server = value
		default:
			return nil, fmt.Errorf("unknown debugviz:app field %q", key)
		}
	}
	return ann, nil
}

func hasWireSkipComment(cg *ast.CommentGroup, kind string) bool {
	if cg == nil {
		return false
	}
	for _, c := range cg.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		switch kind {
		case "file":
			if text == "debugviz:wire skip" {
				return true
			}
		case "http":
			if text == "debugviz:wire http skip" || text == "debugviz:wire skip" {
				return true
			}
		case "configure":
			if text == "debugviz:wire configure skip" || text == "debugviz:wire skip" {
				return true
			}
		}
	}
	return false
}

func fileHasWireSkip(file *ast.File) bool {
	if file == nil {
		return false
	}
	for _, cg := range file.Comments {
		if hasWireSkipComment(cg, "file") {
			return true
		}
	}
	return false
}

func serviceNameExpr(cfg Config, ann *AppAnnotation, localIdent string) ast.Expr {
	name := cfg.ServiceName
	if ann != nil && ann.Name != "" {
		name = ann.Name
	}
	if localIdent != "" {
		return ast.NewIdent(localIdent)
	}
	if name != "" {
		return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", name)}
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("os"),
			Sel: ast.NewIdent("Getenv"),
		},
		Args: []ast.Expr{
			&ast.BasicLit{Kind: token.STRING, Value: `"DEBUGVIZ_SERVICE_NAME"`},
		},
	}
}
