package container

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// exposed is a use case that takes a collaborator at construction and leaves
// the field holding it open to be written afterwards.
type exposed struct {
	in     string
	fields []string
}

// getExposedDependencies are the types of one file whose constructor asks for a
// collaborator that the type then carries in an exported field.
func getExposedDependencies(file *ast.File) []exposed {
	// A constructor is what a New function answers with.
	ctors := map[string][]string{}
	ast.Inspect(file, func(node ast.Node) bool {
		fn, is := node.(*ast.FuncDecl)
		if !is || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "New") ||
			fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
			return true
		}
		made := getTypeName(fn.Type.Results.List[0].Type)
		if made == "" {
			return true
		}
		var taken []string
		for _, one := range fn.Type.Params.List {
			for _, name := range one.Names {
				taken = append(taken, strings.ToLower(name.Name))
			}
		}
		ctors[strings.TrimPrefix(made, "*")] = taken
		return true
	})

	var found []exposed
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		held, is := spec.Type.(*ast.StructType)
		if !is {
			return true
		}
		taken, made := ctors[spec.Name.Name]
		if !made {
			return true
		}
		var open []string
		for _, one := range held.Fields.List {
			for _, name := range one.Names {
				if name.IsExported() && slices.Contains(taken, strings.ToLower(name.Name)) {
					open = append(open, name.Name)
				}
			}
		}
		if open != nil {
			found = append(found, exposed{spec.Name.Name, open})
		}
		return true
	})
	return found
}

// getUseCases are every use case of the core, under the file it stands in.
func getUseCases(t *testing.T) []string {
	t.Helper()
	var said []string
	held, err := os.ReadDir(filepath.Join("..", "usecase"))
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range held {
		if !folder.IsDir() {
			continue
		}
		at := filepath.Join("..", "usecase", folder.Name())
		files, err := os.ReadDir(at)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range files {
			if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
				strings.HasSuffix(one.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(at, one.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, found := range getExposedDependencies(file) {
				said = append(said, fmt.Sprintf("usecase/%s/%s %s exposes %s",
					folder.Name(), one.Name(), found.in, strings.Join(found.fields, ", ")))
			}
		}
	}
	return said
}

// A use case is made with everything it needs, or it is not made.
func TestNoUseCaseCarriesACollaboratorInTheOpen(t *testing.T) {
	for _, one := range getUseCases(t) {
		t.Error(one + ": a use case is made with what it needs, and not given it afterwards")
	}
}

// What the rule refuses, read against a file written to be refused: a field the
// constructor also takes, and not the ones it does not.
func TestWhatTheUseCaseRuleRefuses(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "held.go", `package p

type Read struct {
	Readers port.VaultReaders
	Pages   int
	notes   port.NoteQueries
}

func NewRead(readers port.VaultReaders, notes port.NoteQueries) Read {
	return Read{Readers: readers, notes: notes}
}

type Write struct {
	Writers port.VaultWriters
}

type Search struct {
	Passages port.PassageQueries
}

func NewOver(passages port.PassageQueries) Search {
	return Search{Passages: passages}
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var refused []string
	for _, one := range getExposedDependencies(file) {
		refused = append(refused, one.in+": "+strings.Join(one.fields, ", "))
	}
	// Pages is the caller's own and no collaborator; notes is closed; Write has
	// no constructor to promise anything. Search is reached through a
	// constructor not named after it.
	slices.Sort(refused)
	if !slices.Equal(refused, []string{"Read: Readers", "Search: Passages"}) {
		t.Errorf("the rule refuses %v, want the one collaborator left in the open", refused)
	}
}
