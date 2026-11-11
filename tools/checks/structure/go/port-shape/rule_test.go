package portshape

import (
	"go/ast"
	"path"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getInterfaceProbes(file *ast.File) []string {
	var found []string
	probe := func(target ast.Expr) {
		switch targetVal := target.(type) {
		case *ast.InterfaceType:
			found = append(found, "interface{...}")
		case *ast.SelectorExpr:
			if from, is := targetVal.X.(*ast.Ident); is && load.IsPortName(from.Name) {
				found = append(found, from.Name+"."+targetVal.Sel.Name)
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch targetItem := node.(type) {
		case *ast.TypeAssertExpr:
			if targetItem.Type != nil {
				probe(targetItem.Type)
			}
		case *ast.CaseClause:
			for _, one := range targetItem.List {
				probe(one)
			}
		}
		return true
	})
	return found
}

func internalTypes(file *ast.File) []string {
	local := map[string]bool{}
	for _, one := range file.Imports {
		target, err := strconv.Unquote(one.Path.Value)
		if err != nil || !strings.HasPrefix(target, load.Module+"internal/") {
			continue
		}
		name := path.Base(target)
		if one.Name != nil {
			name = one.Name.Name
		}
		local[name] = true
	}
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		if from, is := pos.X.(*ast.Ident); is && local[from.Name] {
			found = append(found, from.Name+"."+pos.Sel.Name)
		}
		return true
	})
	return found
}
