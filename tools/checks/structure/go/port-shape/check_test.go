package portshape

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestNoCodeOfTheCoreProbesAValueForAnotherInterface checks that a value is
// used as the port it was received as. Probing for another type creates a
// second port that no constructor, composition root or fake references.
// A case clause of an expression switch is checked as well: it names a port
// only where the switch is over a type.
func TestNoCodeOfTheCoreProbesAValueForAnotherInterface(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(pos string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(pos, ".go") {
			return err
		}
		pkg := load.GetPackage("..", pos)
		if strings.HasSuffix(pos, "_test.go") || load.IsAdapter(pkg) || load.IsShared(pkg) || load.IsMachinery(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), pos, nil, 0)
		if err != nil {
			return err
		}
		read++
		for _, one := range getInterfaceProbes(file) {
			wrong = append(wrong, pos+" asserts to "+one)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": a value is used as the port it was handed as")
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
}

// assertion to an anonymous interface and to a port, in a type switch as well,
// and lets through an assertion to a concrete type and to an error.
func TestInterfaceProbeRule_RejectsInterfaceAssertions(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "probe.go", `package p

func probe(target any, reader vaultport.VaultReader) {
	_, _ = target.(interface{ Concurrency() int })
	_, _ = reader.(vaultport.SharedOpener)
	switch target.(type) {
	case machineport.Trash:
	case *os.File:
	}
	_, _ = target.(string)
	_, _ = target.(*Embedder)
	_, _ = target.(error)
	switch target {
	case 1:
	}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"interface{...}", "vaultport.SharedOpener", "machineport.Trash"}
	if got := getInterfaceProbes(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

// An adapter outside the core cannot name a package under internal/, so a port
// domain/ or beside the port.
func TestNoPortNamesATypeFromInternal(t *testing.T) {
	dir := filepath.Join("..", "port")
	entries, err := load.ReadTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	var read int
	for _, item := range entries {
		if !strings.HasSuffix(item.Path, ".go") || strings.HasSuffix(item.Path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, item.Path), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		read++
		rel := "port/" + item.Path
		for _, one := range internalTypes(file) {
			t.Errorf("%s names %s: a port uses the core own types", rel, one)
		}
	}

	// A walk that read no file of port/ is a rule checked against nothing.
	if read < 20 {
		t.Fatalf("%d files of port/ read: the walk is not reading it", read)
	}
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
		"io/fs":         true,
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
