package index_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var numberedSQL = regexp.MustCompile(`^\d+_.*\.sql$`)

func TestTheSchemaOfTheIndexIsOneFileWithoutNumberedMigrations(t *testing.T) {
	if _, err := os.Stat("schema.sql"); err != nil {
		t.Errorf("the schema is not one file, schema.sql: %v", err)
	}
	if _, err := os.Stat("migration"); err == nil {
		t.Error("the folder migration is still there")
	}
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && numberedSQL.MatchString(d.Name()) {
			t.Errorf("a numbered migration file is still there: %s", path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTheIndexHasNoStepRunnerAndNoBookkeepingTable(t *testing.T) {
	removed := map[string]bool{
		"migration": true, "loadMigrations": true, "parseVersion": true,
		"splitStatements": true, "createBookkeeping": true,
		"resetIndex": true, "dropEverything": true, "getDropping": true,
	}
	fset := token.NewFileSet()
	var files []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "schema_migrations") {
			t.Errorf("%s still names the table schema_migrations", path)
		}
		if filepath.Dir(path) != "." {
			continue
		}
		file, err := parser.ParseFile(fset, path, raw, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && removed[d.Name.Name] {
					t.Errorf("%s still declares the function %s of the step runner", path, d.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && removed[ts.Name.Name] {
						t.Errorf("%s still declares the type %s of the step runner", path, ts.Name.Name)
					}
				}
			}
		}
	}
}
