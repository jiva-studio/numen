package load

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

// Finding is one place a structural rule rejects, by file and line, and the name it rejects there.
type Finding struct {
	Position string
	Name     string
}

func NewFinding(fset *token.FileSet, pos token.Pos, name string) Finding {
	position := fset.Position(pos)
	return Finding{Position: filepath.ToSlash(position.Filename) + ":" + strconv.Itoa(position.Line), Name: name}
}

func SortFindings(found []Finding) []Finding {
	slices.SortFunc(found, func(a, b Finding) int {
		if order := strings.Compare(a.Position, b.Position); order != 0 {
			return order
		}
		return strings.Compare(a.Name, b.Name)
	})
	return slices.Compact(found)
}

func ListProductionPackages(pkgs []*packages.Package) []*packages.Package {
	var found []*packages.Package
	for _, pkg := range pkgs {
		if pkg.Types != nil && pkg.TypesInfo != nil && !IsTestVariant(pkg) {
			found = append(found, pkg)
		}
	}
	return found
}

func GetModulePackages(t *testing.T, workspace WorkspacePackages, rel string) []*packages.Package {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(RepoRootDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	pkgs := workspace.ListModule(t, dir)
	for _, pkg := range pkgs {
		for _, one := range pkg.Errors {
			t.Errorf("%s does not load: %v", pkg.ID, one)
		}
	}
	return ListProductionPackages(pkgs)
}

func WriteTreeFixture(t *testing.T, files map[string]string) string {
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

func LoadFixture(t *testing.T, dir string) []*packages.Package {
	t.Helper()
	pkgs := LoadPackages(t, dir, []string{"GOWORK=off"}, "./...")
	for _, pkg := range pkgs {
		for _, one := range pkg.Errors {
			t.Fatalf("the fixture %s does not load: %v", pkg.ID, one)
		}
	}
	return ListProductionPackages(pkgs)
}

func FormatFindings(found []Finding) []string {
	lines := make([]string, len(found))
	for i, one := range found {
		lines[i] = filepath.Base(one.Position) + " " + one.Name
	}
	slices.Sort(lines)
	return lines
}
