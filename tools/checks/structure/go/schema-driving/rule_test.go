package schemadriving

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// schema is the generated connect package. An adapter that names one of its handler types serves
// something outside the core, whether over the wire or to another adapter, which makes it a driving
// adapter. An adapter that only builds its messages or clients calls out and is driven.
const schema = load.Wire + "/gen/numen/v1/numenv1connect"

func isServingSchema(file *ast.File) bool {
	name := ""
	for _, one := range file.Imports {
		if to, err := strconv.Unquote(one.Path.Value); err == nil && to == schema {
			name = "numenv1connect"
			if one.Name != nil {
				name = one.Name.Name
			}
		}
	}
	if name == "" {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		selector, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		if from, is := selector.X.(*ast.Ident); is && from.Name == name && strings.HasSuffix(selector.Sel.Name, "Handler") {
			found = true
		}
		return true
	})
	return found
}

func findDrivenSchemaServers(root string) ([]string, int, error) {
	var wrong []string
	serving := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := load.GetPackage(root, path)
		if !load.IsAdapter(pkg) && !load.IsShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		if !isServingSchema(file) {
			return nil
		}
		serving++
		if !load.IsDriving(pkg) {
			wrong = append(wrong, load.GetFamily(pkg)+" serves the schema and is named driven")
		}
		return nil
	})
	return wrong, serving, err
}
