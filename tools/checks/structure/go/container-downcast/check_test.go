package containerdowncast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestContainerDoesNotDowncastPluginHost ensures container code interacts with
// plugin supervisor solely through the capabilities of container.PluginHost, never downcasting
// to concrete *plugin.Host.
func TestContainerDoesNotDowncastPluginHost(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		node, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			ta, ok := n.(*ast.TypeAssertExpr)
			if !ok {
				return true
			}
			star, ok := ta.Type.(*ast.StarExpr)
			if !ok {
				return true
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if pkg.Name == "plugin" && sel.Sel.Name == "Host" {
				t.Errorf("%s: downcast to *plugin.Host is forbidden; use container.PluginHost", file)
			}
			return true
		})
	}
}
