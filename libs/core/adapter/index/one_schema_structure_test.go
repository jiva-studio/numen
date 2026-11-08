package index_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoCodeOfTheIndexEmptiesItOrReshapesItsBookkeeping(t *testing.T) {
	removed := map[string]bool{"resetIndex": true, "dropEverything": true, "getDropping": true}
	fset := token.NewFileSet()
	sources := os.DirFS(".")
	err := fs.WalkDir(sources, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		raw, err := fs.ReadFile(sources, path)
		if err != nil {
			return err
		}
		for _, banned := range []string{
			"schema_migrations_next", "pragma_table_info('schema_migrations')",
			"ALTER TABLE schema_migrations", "findDroppableSQL",
		} {
			if strings.Contains(string(raw), banned) {
				t.Errorf("%s still holds %q", path, banned)
			}
		}
		if filepath.Dir(path) != "." {
			return nil
		}
		file, err := parser.ParseFile(fset, path, raw, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil && removed[fn.Name.Name] {
				t.Errorf("%s still declares %s", path, fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
