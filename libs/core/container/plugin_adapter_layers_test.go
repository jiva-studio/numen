package container

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

const pluginRootImportPath = "github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"

// getPluginRootImports are the files under dir, tests excluded, that import the plugin adapter's
// root package. Packages below the root never import it.
func getPluginRootImports(dir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			if to, err := strconv.Unquote(imp.Path.Value); err == nil && to == pluginRootImportPath {
				found = append(found, path)
			}
		}
		return nil
	})
	return found, err
}

func TestNoPluginSubpackageImportsThePluginRoot(t *testing.T) {
	root := "../internal/adapter/plugin"
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		found, err := getPluginRootImports(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range found {
			t.Errorf("%s imports the plugin root package: the root assembles its subpackages, and they never reach back up to it", path)
		}
	}
}

func TestPluginRootImportIsFoundInASubpackage(t *testing.T) {
	dir := t.TempDir()
	source := "package sample\n\nimport _ \"" + pluginRootImportPath + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample_test.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	found, err := getPluginRootImports(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || filepath.Base(found[0]) != "sample.go" {
		t.Errorf("found %v, want only sample.go", found)
	}
}
