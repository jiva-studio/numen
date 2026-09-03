package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// filing is a scanned vault whose files can be moved about, with the index kept
// level as a real caller keeps it.
type filing struct {
	db    *container.Index
	vault domain.Vault
	index func(ctx context.Context, v domain.Vault, paths []string) error
	// readers is every reader this vault is opened through, keeping the tally of
	// files whose bytes were read.
	readers *countingReaders
	// went is every note whoever is drawing was told about, in order.
	went *[]domain.Move
}

func fileable(t *testing.T, notes map[string]string) filing {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db := openIndex(t)
	if _, err := scanner(filesystem.VaultReaders{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	readers := &countingReaders{VaultReaders: filesystem.VaultReaders{}}
	refresh := usecase.Refresh{Readers: readers, Notes: db.Notes()}
	return filing{
		db:      db,
		vault:   v,
		readers: readers,
		went:    &[]domain.Move{},
		index: func(ctx context.Context, v domain.Vault, paths []string) error {
			_, err := refresh.Execute(ctx, v, paths)
			return err
		},
	}
}

// move is the move an installation nobody has configured does: a title and a
// filename kept as one name.
func (f filing) move() usecase.Move { return f.moving(true) }

// apart is the move an installation that has turned the two apart does.
func (f filing) apart() usecase.Move { return f.moving(false) }

func (f filing) moving(kept note.SyncTitleAndFilename) usecase.Move {
	return usecase.Move{
		Writers: filesystem.VaultWriters{},
		Links:   f.db.Links(),
		Known:   f.db.SourcesKnown(),
		Sources: f.db.Sources(),
		Notes: note.Move{
			Readers: f.readers,
			Writers: filesystem.VaultWriters{},
			Links:   f.db.Links(),
			Sources: f.db.Sources(),
			Index:   f.index,
			Sync:    func() note.SyncTitleAndFilename { return kept },
			Moving: func(_ context.Context, went domain.Move) {
				*f.went = append(*f.went, went)
			},
		},
	}
}

// resolves is where the one link written in a note reaches.
func (f filing) resolves(t *testing.T, in string) string {
	t.Helper()
	found, err := f.db.Links().Links(t.Context(), string(f.vault.ID), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("%s holds %d links, want one: %+v", in, len(found), found)
	}
	return found[0].To
}

func (f filing) read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.vault.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

// A folder moves with everything under it, whatever kind of file that is. The
// notes are filed where they now are, and the links written by their names
// follow them.
func TestAFolderMovesWithTheNotesUnderIt(t *testing.T) {
	t.Parallel()
	f := fileable(t, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"physics/Heat.md":    "# Heat\n",
		"ByName.md":          "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# By name\n",
		"ByPath.md":          "---\nlinks:\n  - to: physics/Heat.md\n    role: parent\n---\n# By path\n",
	})
	testsupport.WriteBook(t, f.vault.Path, "physics/A Book.epub")
	byName := f.read(t, "ByName.md")

	moved, err := f.move().Execute(t.Context(), f.vault, "physics", "science/physics")
	if err != nil {
		t.Fatal(err)
	}
	if !moved.Landed {
		t.Fatal("the folder did not land")
	}

	for _, path := range []string{"science/physics/Entropy.md", "science/physics/Heat.md", "science/physics/A Book.epub"} {
		if _, err := os.Stat(filepath.Join(f.vault.Path, filepath.FromSlash(path))); err != nil {
			t.Errorf("%s did not arrive: %v", path, err)
		}
	}

	if got := f.resolves(t, "ByName.md"); got != "science/physics/Entropy.md" {
		t.Errorf("the link written by name reaches %q", got)
	}
	if got := f.read(t, "ByName.md"); got != byName {
		t.Errorf("a link that was not broken was rewritten\n want %q\n  got %q", byName, got)
	}

	if got := f.resolves(t, "ByPath.md"); got != "science/physics/Heat.md" {
		t.Errorf("the link written by the path it had reaches %q", got)
	}
	if len(moved.Repaired) != 0 {
		t.Errorf("a name reaches its note wherever it is, so nothing is repaired: %v", moved.Repaired)
	}
}

// sources is what the index holds at a path and beneath it, by path.
func (f filing) sources(t *testing.T, path string) []string {
	t.Helper()
	found, err := f.db.SourcesKnown().Under(t.Context(), string(f.vault.ID), path)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(found))
	for _, ref := range found {
		out = append(out, ref.Path)
	}
	return out
}

// What the vault holds under a path is asked of the index, and it is the same
// answer walking the folder gives.
func TestWhatIsUnderAPathIsWhatAWalkFinds(t *testing.T) {
	t.Parallel()
	f := fileable(t, map[string]string{
		"physics/Entropy.md":      "# Entropy\n",
		"physics/heat/Heat.md":    "# Heat\n",
		"physics/heat/Carnot.md":  "# Carnot\n",
		"physics-old/Stray.md":    "# Stray\n",
		"chemistry/Reactions.md":  "# Reactions\n",
		".trash/physics/Older.md": "# Older\n",
		"physics/.hidden/Kept.md": "# Kept\n",
		"physics/Notes.txt":       "a list\n",
	})

	// The folder holds the two notes and the folder under it, and neither the
	// hidden folder, the file of no kind, the sibling nor the trash.
	if got := f.sources(t, "physics"); len(got) != 3 {
		t.Fatalf("the folder holds %v, and this test compares what is in it", got)
	}
	for _, path := range []string{"physics", "physics/heat", "physics/Entropy.md", "chemistry"} {
		want := walked(t, f, path)
		if got := f.sources(t, path); !slices.Equal(got, want) {
			t.Errorf("the index holds %v under %s, and a walk finds %v", got, path, want)
		}
	}
}

// walked is every source a walk of the vault reports at a path and beneath it,
// by path.
func walked(t *testing.T, f filing, path string) []string {
	t.Helper()
	reader, err := filesystem.VaultReaders{}.Open(f.vault)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	if err := reader.Walk(t.Context(), func(ref domain.Fingerprint) error {
		if ref.Path == path || strings.HasPrefix(ref.Path, path+"/") {
			out = append(out, ref.Path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

// A folder that travelled is filed where it now is, and what was derived from
// each source is still the source's. The files say what they said, so none of
// them is opened.
func TestAFolderThatMovedIsFiledWhereItIsWithoutBeingRead(t *testing.T) {
	t.Parallel()
	f := fileable(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n\nA measure of disorder.\n",
		"physics/heat/Heat.md": "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
		"physics-old/Stray.md": "# Stray\n",
	})
	f.readers.reads = 0

	if _, err := f.move().Execute(t.Context(), f.vault, "physics", "science/physics"); err != nil {
		t.Fatal(err)
	}

	if read := f.readers.reads; read != 0 {
		t.Errorf("the move read %d files, and a file that moved says what it said", read)
	}

	want := []string{"science/physics/Entropy.md", "science/physics/heat/Heat.md"}
	if got := f.sources(t, "science/physics"); !slices.Equal(got, want) {
		t.Errorf("the index files the folder as %v, want %v", got, want)
	}
	if got := f.sources(t, "physics"); len(got) != 0 {
		t.Errorf("the index still files %v under the folder it left", got)
	}
	if got := f.sources(t, "physics-old"); !slices.Equal(got, []string{"physics-old/Stray.md"}) {
		t.Errorf("a folder whose name begins with the one that moved travelled: %v", got)
	}

	// The chunks are still the source's, so a search answers with the note at
	// the path it is filed under now.
	searching := search.New(f.db.Passages(), filesystem.VaultReaders{}, nil, nil, nil, 0, nil)
	found, err := searching.Execute(t.Context(), f.vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != "science/physics/Entropy.md" {
		t.Errorf("the passages of the note that moved are %+v", found)
	}
	if got := f.resolves(t, "science/physics/heat/Heat.md"); got != "science/physics/Entropy.md" {
		t.Errorf("the link inside the folder reaches %q", got)
	}
}

// A destination that is taken is refused, and nothing has moved when the caller
// is told.
func TestAMoveOntoATakenNameMovesNothing(t *testing.T) {
	t.Parallel()
	f := fileable(t, map[string]string{
		"physics/Entropy.md":     "# Entropy\n",
		"archive/physics/Old.md": "# Old\n",
	})

	moved, err := f.move().Execute(t.Context(), f.vault, "physics", "archive/physics")
	if !errors.Is(err, port.ErrOccupied) {
		t.Fatalf("want ErrOccupied, got %v", err)
	}
	if moved.Landed {
		t.Error("the move says it landed")
	}
	if _, err := os.Stat(filepath.Join(f.vault.Path, "physics", "Entropy.md")); err != nil {
		t.Errorf("the note did not stay where it was: %v", err)
	}
	if got := f.read(t, "archive/physics/Old.md"); got != "# Old\n" {
		t.Errorf("what was already there was written over:\n%s", got)
	}
}

// One file moves by the same use case, and a book is a file like any other.
func TestABookMovesOnItsOwn(t *testing.T) {
	t.Parallel()
	f := fileable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	testsupport.WriteBook(t, f.vault.Path, "A Book.epub")

	if _, err := f.move().Execute(t.Context(), f.vault, "A Book.epub", "library/A Book.epub"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.vault.Path, "library", "A Book.epub")); err != nil {
		t.Errorf("the book did not arrive: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(f.vault.Path, "A Book.epub")); err == nil {
		t.Error("the book is still where it was")
	}
}
