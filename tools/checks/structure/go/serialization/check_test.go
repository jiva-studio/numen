// Package serialization is the check `go/serialization`, run on the tree with the planted violations that prove it.
package serialization

import (
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

)

// TestNoInnerLayerImportsSerializationOrUsesFormatTags verifies that port/, usecase/, and domain/
// import no wire/storage format packages and carry no serialization struct tags.
func TestNoInnerLayerImportsSerializationOrUsesFormatTags(t *testing.T) {
	for _, root := range []string{"../domain", "../port", "../usecase", "../../../note/domain"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			if root != "../domain" {
				for _, imp := range file.Imports {
					to, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						continue
					}
					if getSerializationImports()[to] || strings.HasPrefix(to, "google.golang.org/protobuf/") {
						t.Errorf("%s imports %s: inner layers do not parse wire or storage formats", path, to)
					}
				}
			}
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
					if field.Tag == nil {
						continue
					}
					for _, tag := range listSerializationTags() {
						if strings.Contains(field.Tag.Value, tag) {
							fieldName := "<embedded>"
							if len(field.Names) > 0 {
								fieldName = field.Names[0].Name
							}
							t.Errorf("%s %s.%s has serialization tag %s: format tags belong on adapter DTOs", path, spec.Name.Name, fieldName, field.Tag.Value)
						}
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
