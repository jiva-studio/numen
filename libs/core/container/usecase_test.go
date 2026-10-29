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

// getExposedDependencies returns the types of one file whose constructor requires a
// collaborator that the type then stores in an exported field.
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
		var acquired []string
		for _, one := range fn.Type.Params.List {
			for _, name := range one.Names {
				acquired = append(acquired, strings.ToLower(name.Name))
			}
		}
		ctors[strings.TrimPrefix(made, "*")] = acquired
		return true
	})

	var found []exposed
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		target, is := spec.Type.(*ast.StructType)
		if !is {
			return true
		}
		acquiredVal, made := ctors[spec.Name.Name]
		if !made {
			return true
		}
		var open []string
		for _, one := range target.Fields.List {
			for _, name := range one.Names {
				if name.IsExported() && slices.Contains(acquiredVal, strings.ToLower(name.Name)) {
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

func getUseCases(t *testing.T) []string {
	t.Helper()
	var findings []string
	target, err := os.ReadDir(filepath.Join("..", "usecase"))
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range target {
		if !folder.IsDir() {
			continue
		}
		pos := filepath.Join("..", "usecase", folder.Name())
		files, err := os.ReadDir(pos)
		if err != nil {
			t.Fatal(err)
		}
		for _, one := range files {
			if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") ||
				strings.HasSuffix(one.Name(), "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(pos, one.Name()), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, found := range getExposedDependencies(file) {
				findings = append(findings, fmt.Sprintf("usecase/%s/%s %s exposes %s",
					folder.Name(), one.Name(), found.in, strings.Join(found.fields, ", ")))
			}
		}
	}
	return findings
}

// A use case or service across the repository is made with everything it needs, or it is not made.
func TestNoUseCaseCarriesACollaboratorInTheOpen(t *testing.T) {
	for _, one := range getUseCases(t) {
		t.Error(one + ": a use case is made with what it needs, and not given it afterwards")
	}
	roots := []string{"../adapter", "../internal", "../../notes", "../../../plugins"}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			if strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, ".connect.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, found := range getExposedDependencies(file) {
				t.Errorf("%s %s exposes constructor-injected field(s) %s", path, found.in, strings.Join(found.fields, ", "))
			}
			return nil
		})
	}
}

// constructor also takes, and not the ones it does not.
func TestUseCaseRule_RejectsUnusedPortFields(t *testing.T) {
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
	var violations []string
	for _, one := range getExposedDependencies(file) {
		violations = append(violations, one.in+": "+strings.Join(one.fields, ", "))
	}
	// Pages is the caller's own and no collaborator; notes is closed; Write has
	// no constructor to promise anything. Search is reached through a
	// constructor not named after it.
	slices.Sort(violations)
	if !slices.Equal(violations, []string{"Read: Readers", "Search: Passages"}) {
		t.Errorf("the rule rejects %v, want the one collaborator left in the open", violations)
	}
}
