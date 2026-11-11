// Package coretestsuseexternalpackages is the check `go/core-tests-use-external-packages`, run on the tree with the planted violations that prove it.
package coretestsuseexternalpackages

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestCoreTestsBuildingAdaptersUseExternalPackages checks that a test building an adapter is
// external to the package it exercises, because an adapter built from within the package is in its
// compilation unit. The rules above skip test files for that reason.
// An adapter's own test is not covered: it builds the adapter under test, as does the composition
// root's.
func TestCoreTestsBuildingAdaptersUseExternalPackages(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := load.GetPackage("..", path)
		if load.IsAdapter(pkg) || load.IsUnder(pkg, "container") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		if strings.HasSuffix(file.Name.Name, "_test") {
			return nil
		}
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil || !strings.HasPrefix(to, load.Module) {
				continue
			}
			if target := strings.TrimPrefix(to, load.Module); load.IsAdapter(target) {
				wrong = append(wrong, path+" builds "+target+" from inside "+pkg)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": a test that builds an adapter stands outside the package")
	}

	// A walk that read no test of the core is a rule checked against nothing.
	if read < 100 {
		t.Fatalf("%d tests of the core read: the walk is not reading them", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
