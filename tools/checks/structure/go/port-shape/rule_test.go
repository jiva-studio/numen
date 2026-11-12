package portshape

import (
	"go/ast"
	"go/types"
	"path"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

func isPortInterface(t types.Type, root string) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || !types.IsInterface(named) {
		return false
	}
	return load.IsInLayer(named.Obj().Pkg().Path(), root, "port")
}

func isProbe(source, target types.Type, root string) bool {
	if target == nil || !types.IsInterface(target) || isPortInterface(target, root) {
		return false
	}
	_, isLiteral := types.Unalias(target).(*types.Interface)
	return isLiteral || source != nil && isPortInterface(source, root)
}

func getInterfaceProbes(pkg *packages.Package, root string) []load.Finding {
	var found []load.Finding
	probe := func(pos ast.Node, subject, target ast.Expr) {
		source, targetType := pkg.TypesInfo.TypeOf(subject), pkg.TypesInfo.TypeOf(target)
		if isProbe(source, targetType, root) {
			found = append(found, load.NewFinding(pkg.Fset, pos.Pos(), "asserts "+source.String()+" to "+targetType.String()))
		}
	}
	for _, file := range pkg.Syntax {
		if ast.IsGenerated(file) {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch item := node.(type) {
			case *ast.TypeAssertExpr:
				if item.Type != nil {
					probe(item, item.X, item.Type)
				}
			case *ast.TypeSwitchStmt:
				subject := getSwitchSubject(item)
				if subject == nil {
					return true
				}
				for _, statement := range item.Body.List {
					clause, ok := statement.(*ast.CaseClause)
					if !ok {
						continue
					}
					for _, one := range clause.List {
						probe(one, subject, one)
					}
				}
			}
			return true
		})
	}
	return found
}

func getSwitchSubject(item *ast.TypeSwitchStmt) ast.Expr {
	var guard ast.Expr
	switch assign := item.Assign.(type) {
	case *ast.ExprStmt:
		guard = assign.X
	case *ast.AssignStmt:
		if len(assign.Rhs) == 1 {
			guard = assign.Rhs[0]
		}
	}
	if assertion, ok := guard.(*ast.TypeAssertExpr); ok {
		return assertion.X
	}
	return nil
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
