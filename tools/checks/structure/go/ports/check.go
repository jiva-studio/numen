package ports

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// An adapter the compiler holds is one an application cannot name. Where the
// composition root answers with one, the application holds a value it has no
// word for: it cannot declare a variable of that type, cannot write a stand-in
// for it in a test, and cannot see in its own source what it was given.
//
// A driven adapter therefore is returned as the port it satisfies. A driving one
// is what an application mounts and calls in through, and has no port to come
// back as; those are named in driving and are not this rule's business.
const compilerHolds = load.Module + "internal/adapter/"

func getRetainedAdapters(name, src string) []string {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
	if err != nil {
		return []string{name + " does not parse: " + err.Error()}
	}

	underMap := map[string]string{}
	for _, one := range f.Imports {
		path, err := strconv.Unquote(one.Path.Value)
		if err != nil {
			continue
		}
		pos := path[strings.LastIndex(path, "/")+1:]
		if one.Name != nil {
			pos = one.Name.Name
		}
		underMap[pos] = path
	}

	var findings []string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() || fn.Type.Results == nil || !isReachable(fn) {
			continue
		}
		for _, result := range fn.Type.Results.List {
			ast.Inspect(result.Type, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				offset, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				path := underMap[offset.Name]
				if !strings.HasPrefix(path, compilerHolds) || load.GetDriving()[strings.TrimPrefix(path, load.Module)] {
					return true
				}
				findings = append(findings, fn.Name.Name+" answers with "+offset.Name+"."+sel.Sel.Name+
					", which is "+strings.TrimPrefix(path, load.Module)+"'s own type and not a port")
				return true
			})
		}
	}
	return findings
}

func isReachable(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return true
	}
	on := fn.Recv.List[0].Type
	if star, ok := on.(*ast.StarExpr); ok {
		on = star.X
	}
	name, ok := on.(*ast.Ident)
	return ok && name.IsExported()
}
