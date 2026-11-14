package scancomplete

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

func isHelperPackage(rel string) bool {
	return load.IsHelper(rel) || load.IsWithin(rel, "internal/adapterlib") || load.IsWithin(rel, "internal/testsupport")
}

func isTestSupport(rel string) bool { return load.IsWithin(rel, "internal/testsupport") }

func isAdapterLib(rel string) bool { return load.IsWithin(rel, "internal/adapterlib") }

func getHelperImportFindings(pkg *packages.Package, root, rel string) []string {
	var found []string
	for path := range pkg.Imports {
		target := strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
		switch {
		case isAdapterLib(rel):
			if load.IsInLayer(path, root, "usecase") {
				found = append(found, rel+" imports "+target+": shared adapter code depends on no use case")
			}
			if load.IsInLayer(path, root, "adapter") || load.IsInLayer(path, root, "internal/adapter") {
				found = append(found, rel+" imports "+target+": shared adapter code depends on no other adapter")
			}
		case !isTestSupport(rel) && (load.IsInLayer(path, root, "port") || load.IsInLayer(path, root, "usecase")):
			found = append(found, rel+" imports "+target+": a helper depends on no port and no use case")
		case load.IsTechnologyImport(path):
			found = append(found, rel+" imports "+path+": a helper reaches no technology")
		}
	}
	return found
}

func getMutatingMethods(pkg *packages.Package) []string {
	var found []string
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Body == nil || len(function.Recv.List[0].Names) == 0 {
				continue
			}
			typeName, isPointer := getReceiverTypeName(function.Recv.List[0].Type)
			if !isPointer || !ast.IsExported(typeName) || !function.Name.IsExported() || !isTypeDeclaredHere(pkg, typeName) {
				continue
			}
			if isMutating(pkg.TypesInfo, function.Body, pkg.TypesInfo.Defs[function.Recv.List[0].Names[0]]) {
				found = append(found, typeName+"."+function.Name.Name)
			}
		}
	}
	slices.Sort(found)
	return slices.Compact(found)
}

func isTypeDeclaredHere(pkg *packages.Package, name string) bool {
	_, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
	return ok
}

func getReceiverTypeName(expr ast.Expr) (name string, isPointer bool) {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr, isPointer = star.X, true
	}
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name, isPointer
	case *ast.IndexExpr:
		if ident, ok := node.X.(*ast.Ident); ok {
			return ident.Name, isPointer
		}
	}
	return "", false
}

func isMutating(info *types.Info, body *ast.BlockStmt, receiver types.Object) bool {
	if receiver == nil {
		return false
	}
	isRooted := func(expr ast.Expr) bool {
		for {
			switch node := ast.Unparen(expr).(type) {
			case *ast.SelectorExpr:
				expr = node.X
			case *ast.IndexExpr:
				expr = node.X
			case *ast.StarExpr:
				expr = node.X
			case *ast.Ident:
				return info.Uses[node] == receiver
			default:
				return false
			}
		}
	}
	isWritten := func(expr ast.Expr) bool {
		if _, isIdent := ast.Unparen(expr).(*ast.Ident); isIdent {
			return false
		}
		return isRooted(expr)
	}
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.AssignStmt:
			found = found || slices.ContainsFunc(item.Lhs, isWritten)
		case *ast.IncDecStmt:
			found = found || isWritten(item.X)
		case *ast.CallExpr:
			if name, ok := item.Fun.(*ast.Ident); ok && (name.Name == "delete" || name.Name == "clear") && len(item.Args) > 0 {
				found = found || isWritten(item.Args[0])
			}
		}
		return !found
	})
	return found
}

func getHelperFindings(pkgs []*packages.Package, root string) (found []string, read int) {
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		if !isHelperPackage(rel) {
			continue
		}
		read++
		found = append(found, getHelperImportFindings(pkg, root, rel)...)
		if isTestSupport(rel) || isAdapterLib(rel) {
			continue
		}
		for _, method := range getMutatingMethods(pkg) {
			found = append(found, rel+" exports "+method+", which changes the state of its receiver: a helper holds no state")
		}
	}
	slices.Sort(found)
	return found, read
}
