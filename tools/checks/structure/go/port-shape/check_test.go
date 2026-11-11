// Package portshape is the check `go/port-shape`, run on the tree with the planted violations that prove it.
package portshape

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A value is used as the port it was received as. Probing it for another interface creates a second port
// that no constructor, composition root or fake references, so a type assertion or a type-switch case whose
// target is an interface that no port declares, on a value of a port type, is refused, whether the target is
// a named interface of the package that probes or an interface spelled in place.
func TestNoCodeOfTheCoreProbesAValueForAnotherInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	workspace := load.Workspace(t, "libs/core/kernel")
	pkgs := load.GetModulePackages(t, workspace, "libs/core/kernel")
	var found []load.Finding
	for _, pkg := range pkgs {
		found = append(found, getInterfaceProbes(pkg, load.CoreImportPath)...)
	}
	for _, one := range load.SortFindings(found) {
		t.Errorf("%s: %s: a value is used as the port it was handed as", one.Position, one.Name)
	}
	if len(pkgs) == 0 {
		t.Fatal("no package of the core read: the walk checks nothing")
	}
}

// The rule refuses an assertion of a port value to a named interface of the asserting package, to an
// interface spelled in place and to an interface of a package that is no port, in a type switch as well,
// and lets through an assertion to a port, to a concrete type and a probe of a value that is no port.
func TestInterfaceProbeRule_RejectsInterfaceAssertionsOnPorts(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"port/vaultport/vaultport.go": `package vaultport

type Reader interface{ Read() string }

type Opener interface{ Open() }
`,
		"usecase/probe/probe.go": `package probe

import (
	"io"

	"fixture/port/vaultport"
)

type watcher interface{ OnChange() }

type Locator interface{ Path() string }

func probe(reader vaultport.Reader, anything any, failure error) {
	_, _ = reader.(watcher)
	_, _ = reader.(Locator)
	_, _ = reader.(io.Closer)
	_, _ = reader.(interface{ Concurrency() int })
	switch reader.(type) {
	case watcher:
	case *concrete:
	}
	switch value := reader.(type) {
	case Locator:
		_ = value
	}
	_, _ = reader.(vaultport.Opener)
	_, _ = reader.(*concrete)
	_, _ = anything.(watcher)
	_, _ = anything.(string)
	_, _ = failure.(Locator)
	_, _ = anything.(interface{ Concurrency() int })
}

type concrete struct{}

func (*concrete) Read() string { return "" }
`,
	})
	var got []string
	for _, pkg := range load.LoadFixture(t, dir) {
		got = append(got, load.FormatFindings(getInterfaceProbes(pkg, "fixture"))...)
	}
	want := []string{
		"probe.go:14 asserts fixture/port/vaultport.Reader to fixture/usecase/probe.watcher",
		"probe.go:15 asserts fixture/port/vaultport.Reader to fixture/usecase/probe.Locator",
		"probe.go:16 asserts fixture/port/vaultport.Reader to io.Closer",
		"probe.go:17 asserts fixture/port/vaultport.Reader to interface{Concurrency() int}",
		"probe.go:19 asserts fixture/port/vaultport.Reader to fixture/usecase/probe.watcher",
		"probe.go:23 asserts fixture/port/vaultport.Reader to fixture/usecase/probe.Locator",
		"probe.go:31 asserts any to interface{Concurrency() int}",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule refuses\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// An adapter outside the core cannot name a package under internal/, so a port
// domain/ or beside the port.
func TestNoPortNamesATypeFromInternal(t *testing.T) {
	wrong, err := findInternalPortTypes(filepath.Join("..", "port"))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// TestPortWalk_FindsThePlantedInternalType verifies that the walk finds a port that names a type of
// a package under internal/, and fails on a folder without a port file.
func TestPortWalk_FindsThePlantedInternalType(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"vaultport/vaultport.go": "package vaultport\n\nimport \"" + load.Module + "internal/highlight\"\n\ntype Reader interface{ Read() highlight.Span }\n",
		"noteport/noteport.go":   "package noteport\n\ntype Reader interface{ Read() string }\n",
	})
	wrong, err := findInternalPortTypes(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "vaultport.go names highlight.Span") {
		t.Errorf("the walk found %v, want highlight.Span in vaultport.go only", wrong)
	}
	if _, err := findInternalPortTypes(t.TempDir()); err == nil {
		t.Error("a folder without a port file passed")
	}
}

func findInternalPortTypes(dir string) ([]string, error) {
	// Fails when it reads no file.
	entries, err := load.ReadTree(dir)
	if err != nil {
		return nil, err
	}
	var wrong []string
	var read int
	for _, item := range entries {
		if !strings.HasSuffix(item.Path, ".go") || strings.HasSuffix(item.Path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, item.Path), nil, 0)
		if err != nil {
			return nil, err
		}
		read++
		rel := "port/" + item.Path
		for _, one := range internalTypes(file) {
			wrong = append(wrong, fmt.Sprintf("%s names %s: a port uses the core own types", rel, one))
		}
	}
	if read == 0 {
		return nil, fmt.Errorf("no file read under %s: the walk checks nothing", dir)
	}
	return wrong, nil
}

// type of a package under internal/, by whatever name the file imports it, and
// lets through the domain's, the standard library's and a selector on a value.
func TestPortInternalRule_RejectsInternalTypes(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "port.go", `package port

import (
	"image"

	"`+load.Module+`domain"
	"`+load.Module+`internal/ocrartifact"
	hl "`+load.Module+`internal/highlight"
)

type Reader interface {
	Read(page image.Image) ([]ocrartifact.Block, error)
	Boxes() []hl.Box
	Range() domain.ByteRange
	Bounds(page image.Image) image.Rectangle
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ocrartifact.Block", "hl.Box"}
	if got := internalTypes(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// TestPortsAreIndependentAndPure verifies that no port/* package imports another port/* package,
// and no port/* package imports infrastructure/OS packages (net/url, encoding/json, path/filepath, io/fs, os).
func TestPortsAreIndependentAndPure(t *testing.T) {
	dir := filepath.Join("..", "port")
	entries, err := load.ReadTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	forbiddenStd := map[string]bool{
		"net/url":       true,
		"encoding/json": true,
		"path/filepath": true,
		"os":            true,
	}
	for _, item := range entries {
		if !strings.HasSuffix(item.Path, ".go") || strings.HasSuffix(item.Path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, item.Path), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		rel := "port/" + item.Path
		for _, imp := range file.Imports {
			target, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			if strings.HasPrefix(target, load.Module+"port/") {
				t.Errorf("%s imports sibling port %s: port packages must be independent contracts", rel, strings.TrimPrefix(target, load.Module))
			}
			if forbiddenStd[target] {
				t.Errorf("%s imports %s: port packages must use pure domain value objects", rel, target)
			}
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
