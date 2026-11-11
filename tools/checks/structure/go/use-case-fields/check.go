package usecasefields

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// exposed is a use case that takes a collaborator at construction and leaves
// the field holding it open to be written afterwards.
type exposed struct {
	in     string
	fields []string
}

func getExposedDependencies(file *ast.File) []exposed {
	// A constructor is what a New function answers with.
	ctors := map[string][]string{}
	ast.Inspect(file, func(node ast.Node) bool {
		fn, is := node.(*ast.FuncDecl)
		if !is || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "New") ||
			fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
			return true
		}
		made := load.GetTypeName(fn.Type.Results.List[0].Type)
		if made == "" {
			return true
		}
		var acquired []string
		for _, one := range fn.Type.Params.List {
			for _, name := range one.Names {
				acquired = append(acquired, strings.ToLower(name.Name))
			}
		}
		ctors[strings.TrimPrefix(made, "*")] = acquired
		return true
	})

	var found []exposed
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		target, is := spec.Type.(*ast.StructType)
		if !is {
			return true
		}
		acquiredVal, made := ctors[spec.Name.Name]
		if !made {
			return true
		}
		var open []string
		for _, one := range target.Fields.List {
			for _, name := range one.Names {
				if name.IsExported() && slices.Contains(acquiredVal, strings.ToLower(name.Name)) {
					open = append(open, name.Name)
				}
			}
		}
		if open != nil {
			found = append(found, exposed{spec.Name.Name, open})
		}
		return true
	})
	return found
}

func getUseCases(t *testing.T) []string {
	t.Helper()
	var findings []string
	target, err := os.ReadDir(filepath.Join("..", "usecase"))
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range target {
		if !folder.IsDir() {
			continue
		}
		pos := filepath.Join("..", "usecase", folder.Name())
		files, err := os.ReadDir(pos)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range files {
			if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
				strings.HasSuffix(one.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pos, one.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, found := range getExposedDependencies(file) {
				findings = append(findings, fmt.Sprintf("usecase/%s/%s %s exposes %s",
					folder.Name(), one.Name(), found.in, strings.Join(found.fields, ", ")))
			}
		}
	}
	return findings
}
