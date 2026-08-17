package filesystem_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// writing opens an empty vault for writing.
func writing(t *testing.T) (port.VaultWriter, string) {
	t.Helper()
	root := t.TempDir()
	w, err := filesystem.Writers{}.Open(domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}
	return w, root
}

// A folder something else reads is a folder that acts on what it finds. An agent
// working the vault names where a note goes, and a note that lands in one of
// those is a file the index does not know about, that nothing on screen shows,
// and that another program obeys.
func TestANoteCannotBeMovedIntoAnotherToolsFolder(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "SKILL.md", []byte("# Skill\n")); err != nil {
		t.Fatal(err)
	}

	for _, to := range []string{
		".claude/skills/evil/SKILL.md",
		".claude/agents/evil.md",
		".git/hooks/SKILL.md",
		".obsidian/SKILL.md",
		".trash/../.claude/planted.md",
	} {
		if err := w.Move(ctx, "SKILL.md", to); err == nil {
			t.Errorf("moving to %s was allowed", to)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(to))); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s was written: %v", to, err)
		}
	}
}

// Where a note goes when it is taken out of the vault's sight is deliberately not
// a note-place, and moving one there is what removal does.
func TestANoteCanStillBeMovedOutOfSight(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "Entropy.md", ".trash/Entropy.md"); err != nil {
		t.Fatalf("a note could not be taken out of sight: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".trash", "Entropy.md")); err != nil {
		t.Errorf("the note is not in the trash: %v", err)
	}
}

// A note moved to a place a note may live is an ordinary move.
func TestANoteMovesWhereANoteMayLive(t *testing.T) {
	w, root := writing(t)
	ctx := t.Context()

	if err := w.Create(ctx, "Entropy.md", []byte("# Entropy\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, "Entropy.md", "physics/Entropy.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "physics", "Entropy.md")); err != nil {
		t.Errorf("the note did not move: %v", err)
	}
}
