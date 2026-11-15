package onnx

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const repositoryRoot = "github.com/jiva-studio/numen/"

// The module imports no other module of this repository, the plugin SDK included: neither its go.mod
// nor any of its Go files, tests among them, names one.
func TestTheModuleImportsNothingOfTheRepository(t *testing.T) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	var self string
	for i, line := range strings.Split(string(mod), "\n") {
		if path, ok := strings.CutPrefix(line, "module "); ok {
			self = strings.TrimSpace(path)
			continue
		}
		if strings.Contains(line, repositoryRoot) {
			t.Errorf("go.mod line %d names the repository: %s", i+1, strings.TrimSpace(line))
		}
	}
	fset := token.NewFileSet()
	files := 0
	err = filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		files++
		for _, spec := range file.Imports {
			imported, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if strings.HasPrefix(imported, repositoryRoot) && imported != self && !strings.HasPrefix(imported, self+"/") {
				t.Errorf("%s imports %s", path, imported)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("no Go file found: a rule checked against nothing")
	}
}
