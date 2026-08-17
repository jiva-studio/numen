package filesystem_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
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

// A note kept as a link to another file in the vault is a link after a write,
// and the file at the other end holds the new bytes.
func TestWritingThroughALinkLeavesTheLink(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"notes/Real.md": "# Entropy\n"})
	link := filepath.Join(v.Path, "Entropy.md")
	if err := os.Symlink(filepath.Join("notes", "Real.md"), link); err != nil {
		t.Fatal(err)
	}

	writer, err := (filesystem.Writers{}).Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(t.Context(), "Entropy.md", []byte("# Entropy\n\nMine.\n"), domain.FileRef{}); err != nil {
		t.Fatal(err)
	}

	at, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if at.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was replaced by a file of its own")
	}
	body, err := os.ReadFile(filepath.Join(v.Path, "notes", "Real.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "# Entropy\n\nMine.\n" {
		t.Errorf("the note the link points at was not written:\n%s", body)
	}
}

// What a vault leaves alone is answered for out of the file's metadata. The
// file here is sixteen megabytes and cannot be opened at all, and the answer
// still comes back.
func TestAskingAboutAFileTheVaultLeavesAloneOpensNothing(t *testing.T) {
	root := t.TempDir()
	export, err := os.OpenFile(filepath.Join(root, "export.pdf"), os.O_CREATE|os.O_WRONLY, 0o000)
	if err != nil {
		t.Fatal(err)
	}
	if err := export.Truncate(16 << 20); err != nil {
		t.Fatal(err)
	}
	if err := export.Close(); err != nil {
		t.Fatal(err)
	}

	reader, err := filesystem.Open(root, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	switch _, err := reader.Stat(t.Context(), "export.pdf"); {
	case !errors.Is(err, port.ErrNotANote):
		t.Errorf("want ErrNotANote, got %v", err)
	case errors.Is(err, fs.ErrNotExist):
		t.Errorf("a file that is there was called missing: %v", err)
	}

	// Nothing at a path is the other answer, and it is not this one.
	switch _, err := reader.Stat(t.Context(), "Entropy.md"); {
	case !errors.Is(err, fs.ErrNotExist):
		t.Errorf("want fs.ErrNotExist, got %v", err)
	case errors.Is(err, port.ErrNotANote):
		t.Errorf("a path with nothing at it was called a file: %v", err)
	}
}

// One vault is held by one writer at a time, and holding one says nothing
// about the next.
func TestOneVaultIsHeldByOneWriterAtATime(t *testing.T) {
	writers := filesystem.Writers{}
	here := testsupport.NewVault(t, nil)
	elsewhere := testsupport.NewVault(t, nil)

	release, err := writers.Hold(t.Context(), here)
	if err != nil {
		t.Fatal(err)
	}

	waiting, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	if _, err := writers.Hold(waiting, here); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a second hold on a held vault: %v", err)
	}
	if other, err := writers.Hold(t.Context(), elsewhere); err != nil {
		t.Fatalf("another vault was held with this one: %v", err)
	} else {
		other()
	}

	release()
	again, err := writers.Hold(t.Context(), here)
	if err != nil {
		t.Fatal(err)
	}
	again()
}
