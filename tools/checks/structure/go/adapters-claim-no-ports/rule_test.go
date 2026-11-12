package adaptersclaimnoports

import (
	"go/ast"
	"go/token"
	"slices"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getClaimedPorts(file *ast.File) []string {
	var target []string
	ast.Inspect(file, func(node ast.Node) bool {
		decl, is := node.(*ast.GenDecl)
		if !is || decl.Tok != token.VAR {
			return true
		}
		for _, spec := range decl.Specs {
			named, is := spec.(*ast.ValueSpec)
			if !is || !slices.ContainsFunc(named.Names, func(pos *ast.Ident) bool {
				return pos.Name == "_"
			}) {
				continue
			}
			if offset, is := named.Type.(*ast.SelectorExpr); is {
				if from, is := offset.X.(*ast.Ident); is && load.IsPortName(from.Name) {
					target = append(target, offset.Sel.Name)
				}
			}
		}
		return true
	})
	return target
}
