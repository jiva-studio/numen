package schemadriving

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestEveryAdapterServingTheSchemaIsDriving ensures adapters serving or mounting
// generated protocol handlers are declared as driving adapters.
func TestEveryAdapterServingTheSchemaIsDriving(t *testing.T) {
	var wrong []string
	var found int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := load.GetPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || !load.IsAdapter(pkg) && !load.IsShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		if !load.NamesHandler(file) {
			return nil
		}
		found++
		if !load.IsDriving(pkg) {
			wrong = append(wrong, load.GetFamily(pkg)+" serves the schema and is named driven")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if found == 0 {
		t.Fatal("no adapter names the schema: the walk is not reading the adapters")
	}
}
