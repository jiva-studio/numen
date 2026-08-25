package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// filing is a scanned vault whose files can be moved about, with the index kept
// level as a real caller keeps it.
type filing struct {
	db    *container.Index
	vault domain.Vault
	index func(ctx context.Context, v domain.Vault, paths []string) error
}

func fileable(t *testing.T, notes map[string]string) filing {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db := openIndex(t)
	if _, err := scanner(filesystem.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	refresh := usecase.Refresh{Readers: filesystem.Readers{}, Notes: db.Notes()}
	return filing{
		db:    db,
		vault: v,
		index: func(ctx context.Context, v domain.Vault, paths []string) error {
			_, err := refresh.Execute(ctx, v, paths)
			return err
		},
	}
}

func (f filing) move() usecase.Move {
	return usecase.Move{
		Readers: filesystem.Readers{},
		Writers: filesystem.Writers{},
		Links:   f.db.Links(),
		Index:   f.index,
		Notes: note.Move{
			Readers: filesystem.Readers{},
			Writers: filesystem.Writers{},
			Links:   f.db.Links(),
			Index:   f.index,
		},
	}
}

// resolves is where the one link written in a note reaches.
func (f filing) resolves(t *testing.T, in string) string {
	t.Helper()
	found, err := f.db.Links().Links(t.Context(), f.vault.ID, in)
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
	if len(moved.Retargeted) != 0 {
		t.Errorf("nothing was retargeted: %+v", moved.Retargeted)
	}
}

// A destination that is taken is refused, and nothing has moved when the caller
// is told.
func TestAMoveOntoATakenNameMovesNothing(t *testing.T) {
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
