package containerdowncast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func findPluginDowncasts(dir string) ([]string, int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, 0, err
	}
	var wrong []string
	read := 0
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			return nil, read, err
		}
		read++
		ast.Inspect(node, func(n ast.Node) bool {
			assertion, ok := n.(*ast.TypeAssertExpr)
			if !ok {
				return true
			}
			star, ok := assertion.Type.(*ast.StarExpr)
			if !ok {
				return true
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "plugin" {
				wrong = append(wrong, filepath.Base(file)+": assertion to *plugin."+sel.Sel.Name)
			}
			return true
		})
	}
	return wrong, read, nil
}
