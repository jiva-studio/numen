// Package containerdowncast is the check `go/container-downcast`, run on the tree with the planted violations that prove it.
package containerdowncast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestContainerDoesNotDowncastPluginHost ensures container code reaches the roles of the plugin host
// solely through the capabilities of container.PluginHost and the fields of container.PluginRoles, never by an
// assertion to a concrete role of package plugin (*plugin.Discovery, *plugin.Settings and the like).
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
			if pkg.Name == "plugin" {
				t.Errorf("%s: assertion to *plugin.%s is forbidden; use the capabilities of container.PluginHost", file, sel.Sel.Name)
			}
			return true
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
