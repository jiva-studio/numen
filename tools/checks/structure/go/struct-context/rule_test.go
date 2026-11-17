package structcontext

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func findContextFields(roots []string) ([]string, int, error) {
	var wrong []string
	read := 0
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			if strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, ".connect.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			read++
			ast.Inspect(file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := spec.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					return true
				}
				for _, field := range st.Fields.List {
					if load.GetTypeName(field.Type) == "context.Context" {
						fieldName := "<embedded>"
						if len(field.Names) > 0 {
							fieldName = field.Names[0].Name
						}
						wrong = append(wrong, path+" "+spec.Name.Name+"."+fieldName+" holds context.Context in a struct field (pass context per method call instead)")
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return nil, read, err
		}
	}
	return wrong, read, nil
}
