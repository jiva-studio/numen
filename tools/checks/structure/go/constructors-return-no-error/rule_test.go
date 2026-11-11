package constructorsreturnnoerror

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func listFallibleConstructors(fset *token.FileSet, file *ast.File) []string {
	var found []string
	for _, declaration := range file.Decls {
		function, is := declaration.(*ast.FuncDecl)
		if !is || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "New") || function.Type.Results == nil {
			continue
		}
		for _, result := range function.Type.Results.List {
			if name, is := result.Type.(*ast.Ident); is && name.Name == "error" {
				found = append(found, fmt.Sprintf("%s:%d", filepath.Base(fset.Position(function.Pos()).Filename), fset.Position(function.Pos()).Line))
			}
		}
	}
	return found
}

func getFallibleConstructors(t *testing.T, roots ...string) (findings []string, files int) {
	t.Helper()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}
			if ast.IsGenerated(file) {
				return nil
			}
			files++
			for _, one := range listFallibleConstructors(fset, file) {
				findings = append(findings, filepath.Join(filepath.Dir(path), one))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return findings, files
}
