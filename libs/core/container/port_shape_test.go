package container

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// getInterfaceProbes are the type assertions and type-switch cases of a file
// whose target is an interface literal or a port.X, each written as the target.
func getInterfaceProbes(file *ast.File) []string {
	var found []string
	probe := func(target ast.Expr) {
		switch targetVal := target.(type) {
		case *ast.InterfaceType:
			found = append(found, "interface{...}")
		case *ast.SelectorExpr:
			if from, is := targetVal.X.(*ast.Ident); is && isPortName(from.Name) {
				found = append(found, from.Name+"."+targetVal.Sel.Name)
			}
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch targetItem := node.(type) {
		case *ast.TypeAssertExpr:
			if targetItem.Type != nil {
				probe(targetItem.Type)
			}
		case *ast.CaseClause:
			for _, one := range targetItem.List {
				probe(one)
			}
		}
		return true
	})
	return found
}

// A value the core is handed is used as the port it was handed as. Asking it
// whether it is also something else is a second port that no constructor, no
// composition root and no fake is told about, so what a port promises is the
// whole of what the core may rely on.
//
// A case clause of an expression switch is read as well: it names a port only
// where the switch is over a type.
func TestNoCodeOfTheCoreProbesAValueForAnotherInterface(t *testing.T) {
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(pos string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(pos, ".go") {
			return err
		}
		pkg := getPackage("..", pos)
		if strings.HasSuffix(pos, "_test.go") || isAdapter(pkg) || isShared(pkg) || isMachinery(pkg) {
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
func TestWhatTheInterfaceProbeRuleRefuses(t *testing.T) {
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

// internalTypes are the types a file names from a package under internal/, each
// written pkg.Name as the file spells it.
func internalTypes(file *ast.File) []string {
	local := map[string]bool{}
	for _, one := range file.Imports {
		target, err := strconv.Unquote(one.Path.Value)
		if err != nil || !strings.HasPrefix(target, module+"internal/") {
			continue
		}
		name := path.Base(target)
		if one.Name != nil {
			name = one.Name.Name
		}
		local[name] = true
	}
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		if from, is := pos.X.(*ast.Ident); is && local[from.Name] {
			found = append(found, from.Name+"."+pos.Sel.Name)
		}
		return true
	})
	return found
}

// An adapter outside the core cannot name a package under internal/, so a port
// domain/ or beside the port.
func TestNoPortNamesATypeFromInternal(t *testing.T) {
	dir := filepath.Join("..", "port")
	entries, err := readTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	var read int
	for _, item := range entries {
		if !strings.HasSuffix(item.name, ".go") || strings.HasSuffix(item.name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, item.name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		read++
		rel := "port/" + item.name
		for _, one := range internalTypes(file) {
			t.Errorf("%s names %s: a port speaks in the core's own types", rel, one)
		}
	}

	// A walk that read no file of port/ is a rule checked against nothing.
	if read < 20 {
		t.Fatalf("%d files of port/ read: the walk is not reading it", read)
	}
}

// type of a package under internal/, by whatever name the file imports it, and
// lets through the domain's, the standard library's and a selector on a value.
func TestWhatThePortInternalRuleRefuses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "port.go", `package port

import (
	"image"

	"`+module+`domain"
	"`+module+`internal/ocrartifact"
	hl "`+module+`internal/highlight"
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
