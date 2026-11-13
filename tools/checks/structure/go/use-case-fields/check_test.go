// Package usecasefields is the check `go/use-case-fields`, run on the tree with the planted violations that prove it.
package usecasefields

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A use case or service across the repository is made with everything it needs, or it is not made.
func TestNoUseCaseCarriesACollaboratorInTheOpen(t *testing.T) {
	for _, one := range getUseCases(t) {
		t.Error(one + ": a use case is made with what it needs, and not given it afterwards")
	}
	roots := []string{"../adapter", "../internal", "../../../note", "../../../../plugins"}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
		if err != nil {
			t.Fatal(err)
		}
	}
}

// constructor also takes, and not the ones it does not.
func TestUseCaseRule_RejectsUnusedPortFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "held.go", `package p

type Read struct {
	Readers port.VaultReaders
	Pages   int
	notes   port.NoteRefs
}

func NewRead(readers port.VaultReaders, notes port.NoteRefs) Read {
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

type Level struct {
	Name  string
	Title string
}

func NewLevel(name string) Level {
	return Level{Name: name, Title: name}
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

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
