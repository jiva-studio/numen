package scancomplete

import (
	"go/ast"
	"go/types"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

func isHelperPackage(rel string, helpers map[string]string) bool {
	_, isListed := helpers[rel]
	return isListed || load.IsWithin(rel, "internal/adapterlib") || load.IsWithin(rel, "internal/testsupport")
}

// isTestSupport reports a package of internal/testsupport: it implements ports and calls use cases by design, so it may
// import them and may hold state; it still reaches no technology.
func isTestSupport(rel string) bool { return load.IsWithin(rel, "internal/testsupport") }

func getHelperImportFindings(pkg *packages.Package, root, rel string) []string {
	var found []string
	for path := range pkg.Imports {
		switch {
		case !isTestSupport(rel) && (load.IsInLayer(path, root, "port") || load.IsInLayer(path, root, "usecase")):
			found = append(found, rel+" imports "+strings.TrimPrefix(strings.TrimPrefix(path, root), "/")+": a helper depends on no port and no use case")
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

func getHelperFindings(pkgs []*packages.Package, root string, helpers map[string]string) (found []string, read int) {
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		if !isHelperPackage(rel, helpers) {
			continue
		}
		read++
		found = append(found, getHelperImportFindings(pkg, root, rel)...)
		if isTestSupport(rel) {
			continue
		}
		for _, method := range getMutatingMethods(pkg) {
			found = append(found, rel+" exports "+method+", which changes the state of its receiver: a helper holds no state")
		}
	}
	slices.Sort(found)
	return found, read
}

var camelBoundary = regexp.MustCompile(`[A-Z][a-z0-9]*|[a-z0-9]+`)

func listPackageWords(pkg *packages.Package) map[string]bool {
	words := map[string]bool{}
	add := func(text string) {
		for _, word := range camelBoundary.FindAllString(text, -1) {
			words[strings.ToLower(word)] = true
		}
	}
	add(pkg.Name)
	for _, file := range pkg.Syntax {
		if file.Doc != nil {
			add(file.Doc.Text())
		}
	}
	for _, name := range pkg.Types.Scope().Names() {
		if ast.IsExported(name) {
			add(name)
		}
	}
	return words
}

func getHelperReasonFindings(pkgs []*packages.Package, root string, helpers map[string]string) []string {
	var found []string
	byRel := map[string]*packages.Package{}
	for _, pkg := range pkgs {
		byRel[strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")] = pkg
	}
	for rel, reason := range helpers {
		pkg, ok := byRel[rel]
		if !ok {
			found = append(found, "the helper "+rel+" is not a package of the kernel")
			continue
		}
		words := listPackageWords(pkg)
		describes := slices.ContainsFunc(strings.FieldsFunc(strings.ToLower(reason), func(r rune) bool { return !unicode.IsLetter(r) }), func(word string) bool {
			return len(word) >= 4 && words[word]
		})
		if !describes {
			found = append(found, "the reason of the helper "+rel+" names nothing the package declares or documents: "+reason)
		}
	}
	slices.Sort(found)
	return found
}
