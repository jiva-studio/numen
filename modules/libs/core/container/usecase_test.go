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

// usecaseBaseline are the use cases still carrying a collaborator in the open,
// each under the file it stands in, and the list only shrinks.
//
// Closing one: the fields go unexported, the constructor refuses what it cannot
// make, and the package's tests build through it. `ProofreadReading` in
// usecase/source is one that is closed.
var usecaseBaseline = []string{
	"usecase/cards/create.go Create exposes Writers, Index, Now",
	"usecase/cards/list.go List exposes Readers, Notes",
	"usecase/cards/read.go Read exposes Readers, Links",
	"usecase/cards/rename.go RenameField exposes Readers, Writers, Notes, Links, Index, Now",
	"usecase/cards/write.go Write exposes Readers, Writers, Links, Index, Now",
	"usecase/flashcards/card_face.go ListCardFaces exposes Readers, Notes, Links",
	"usecase/flashcards/cards_due.go CountCardsDue exposes Schedules, Presets, Day, Now",
	"usecase/flashcards/count_reviews.go CountReviews exposes Logs, Schedules, Day, Now",
	"usecase/flashcards/curve.go ProjectCurve exposes Schedules, Presets, Day, Now",
	"usecase/flashcards/mark_cards.go MarkCards exposes Readers, Writers, Notes, Links, Index, Now",
	"usecase/flashcards/neighbourhood.go ShowNeighbourhood exposes Linked, Notes, Reads",
	"usecase/flashcards/presets.go Presets exposes Readers, Writers, Links, Notes, Index, Day, Now",
	"usecase/flashcards/record.go Record exposes Run, Now",
	"usecase/flashcards/schedules.go Schedules exposes Logs, By, Day, Presets",
	"usecase/flashcards/session.go Session exposes Marks, Schedules, Presets, Day, Now",
	"usecase/note/create.go Create exposes Writers, Names, Index, Now",
	"usecase/note/edit.go Edit exposes Readers, Writers, Index, Now",
	"usecase/note/edit_links.go EditLinks exposes Readers, Writers, Index, Now",
	"usecase/note/move.go Move exposes Readers, Writers, Links, Names, Sources, Index, Now",
	"usecase/note/neighbourhood.go ShowNeighbourhood exposes Links, Notes",
	"usecase/note/read.go Read exposes Readers",
	"usecase/note/remove.go Remove exposes Writers, Links, Queries, Index",
	"usecase/note/replace.go Replace exposes Readers, Writers, Index, Now",
	"usecase/note/show_links.go ShowLinks exposes Links",
	"usecase/note/write.go Write exposes Readers, Writers, Index, Now",
	"usecase/source/embed.go Embed exposes Readers, Chunks, Vectors",
	"usecase/source/extract.go Extract exposes Readers, Sources, Queries",
	"usecase/source/highlight.go Highlight exposes Readers, Sources, Derived",
	"usecase/source/recognise.go Recognise exposes Readers, Sources, Derived, By, Documents",
	"usecase/source/transcribe.go Transcribe exposes Readers, Sources, Derived, By",
	"usecase/vault/add.go Add exposes Identity, Registry, Now",
	"usecase/vault/erase.go Erase exposes Identity, Trash, Forget",
	"usecase/vault/find.go Find exposes Registry",
	"usecase/vault/folder_check.go FolderCheck exposes Readers",
	"usecase/vault/follow.go Follow exposes Watcher, Refresh, Scan",
	"usecase/vault/forget.go Forget exposes Registry, Index",
	"usecase/vault/import.go Import exposes Writers, Files",
	"usecase/vault/known_vaults.go KnownVaults exposes Registry",
	"usecase/vault/list.go List exposes Registry",
	"usecase/vault/move.go Move exposes Writers, Links, Queries, Sources, Notes",
	"usecase/vault/read_whole_vault.go ReadWholeVault exposes Notes, Books, Vectors",
	"usecase/vault/refresh.go Refresh exposes Readers, Vaults, Notes, Queries, Sources",
	"usecase/vault/rename.go Rename exposes Registry, Index",
	"usecase/vault/scan.go Scan exposes Readers, Vaults, Notes, Known, Maintenance",
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
	var wrong []string
	for _, one := range getUseCases(t) {
		if !slices.Contains(usecaseBaseline, one) {
			wrong = append(wrong, one)
		}
	}
	for _, one := range wrong {
		t.Error(one + ": a use case is made with what it needs, and not given it afterwards")
	}
}

// The baseline only shrinks: an entry naming a type that carries no
// collaborator in the open any more is an entry that stayed behind.
func TestNothingIsBaselinedThatNoUseCaseCarriesAnyMore(t *testing.T) {
	said := getUseCases(t)
	for _, one := range usecaseBaseline {
		if !slices.Contains(said, one) {
			t.Error(one + ": closed, and still in the baseline")
		}
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
