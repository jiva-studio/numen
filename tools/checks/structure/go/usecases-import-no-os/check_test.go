package usecasesimportnoos

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestUseCasesImportNoOSFilepathOrFS verifies that usecase/ never imports path/filepath or io/fs.
func TestUseCasesImportNoOSFilepathOrFS(t *testing.T) {
	err := filepath.WalkDir("../usecase", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			to, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			if to == "path/filepath" || to == "io/fs" {
				t.Errorf("%s imports %s: use cases operate on slash-separated vault paths and domain sentinels, not OS filepath or io/fs", path, to)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
