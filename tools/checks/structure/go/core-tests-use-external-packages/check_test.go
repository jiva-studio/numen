// Package coretestsuseexternalpackages is the check `go/core-tests-use-external-packages`, run on the tree with the planted violations that prove it.
package coretestsuseexternalpackages

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
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
	wrong, err := findInternalAdapterTests("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": a test that builds an adapter stands outside the package")
	}
}

// TestInternalTestWalk_FindsThePlantedTest verifies that the walk finds a test inside the package
// that imports an adapter and passes the same test written in the external package.
func TestInternalTestWalk_FindsThePlantedTest(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"usecase/note/inside_test.go":  "package note\n\nimport _ \"" + load.Module + "adapter/index\"\n",
		"usecase/note/outside_test.go": "package note_test\n\nimport _ \"" + load.Module + "adapter/index\"\n",
	})
	wrong, err := findInternalAdapterTests(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "inside_test.go builds adapter/index from inside usecase/note") {
		t.Errorf("the walk found %v, want inside_test.go only", wrong)
	}
}

// TestInternalTestWalk_FailsOnATreeWithoutTests verifies that a root without a test file is an error, not a pass.
func TestInternalTestWalk_FailsOnATreeWithoutTests(t *testing.T) {
	if _, err := findInternalAdapterTests(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a test file passed")
	}
}

func findInternalAdapterTests(root string) ([]string, error) {
	// Fails when it reads no test.
	var wrong []string
	var read int
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := load.GetPackage(root, path)
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
	if err == nil && read == 0 {
		err = fmt.Errorf("no test file read under %s: the walk checks nothing", root)
	}
	return wrong, err
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
