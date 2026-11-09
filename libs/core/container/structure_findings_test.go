package container

import (
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// finding is one place a structural rule rejects, by file and line, and the name it rejects there.
type finding struct {
	position string
	name     string
}

func newFinding(fset *token.FileSet, pos token.Pos, name string) finding {
	position := fset.Position(pos)
	return finding{position: filepath.ToSlash(position.Filename) + ":" + strconv.Itoa(position.Line), name: name}
}

func sortFindings(found []finding) []finding {
	slices.SortFunc(found, func(a, b finding) int {
		if order := strings.Compare(a.position, b.position); order != 0 {
			return order
		}
		return strings.Compare(a.name, b.name)
	})
	return slices.Compact(found)
}

// listProductionPackages returns the variant of each package that holds its non-test files only.
func listProductionPackages(pkgs []*packages.Package) []*packages.Package {
	var found []*packages.Package
	for _, pkg := range pkgs {
		if pkg.Types != nil && pkg.TypesInfo != nil && !isTestVariant(pkg) {
			found = append(found, pkg)
		}
	}
	return found
}

// getModulePackages returns the production packages of the workspace module in the directory under the
// repository root.
func getModulePackages(t *testing.T, workspace workspacePackages, rel string) []*packages.Package {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(repoRootDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	pkgs := workspace.listModule(t, dir)
	for _, pkg := range pkgs {
		for _, one := range pkg.Errors {
			t.Errorf("%s does not load: %v", pkg.ID, one)
		}
	}
	return listProductionPackages(pkgs)
}

// writeTreeFixture writes a module named fixture whose files may lie in folders.
func writeTreeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module fixture\n\ngo 1.26\n"
	for name, text := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// loadFixture loads every package of a fixture module and fails on a fixture that does not compile.
func loadFixture(t *testing.T, dir string) []*packages.Package {
	t.Helper()
	pkgs := loadPackages(t, dir, []string{"GOWORK=off"}, "./...")
	for _, pkg := range pkgs {
		for _, one := range pkg.Errors {
			t.Fatalf("the fixture %s does not load: %v", pkg.ID, one)
		}
	}
	return listProductionPackages(pkgs)
}

// formatFindings writes each finding as the base name of its file, its line and its name, for a fixture.
func formatFindings(found []finding) []string {
	lines := make([]string, len(found))
	for i, one := range found {
		lines[i] = filepath.Base(one.position) + " " + one.name
	}
	slices.Sort(lines)
	return lines
}
