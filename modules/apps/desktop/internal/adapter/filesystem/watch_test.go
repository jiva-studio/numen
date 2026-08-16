package filesystem_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// watching follows a vault and gives back a function that waits for the next
// batch of changed paths.
func watching(t *testing.T, root string) func() []string {
	t.Helper()
	w, err := filesystem.Watch(t.Context(), domain.Vault{Path: root}, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return func() []string {
		select {
		case paths := <-w.Changes:
			slices.Sort(paths)
			return paths
		case <-time.After(5 * time.Second):
			t.Fatal("nothing was reported")
			return nil
		}
	}
}

func write(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestOneSaveIsOneReport. Several events arrive for a save, and the same path
// arrives from several of them; what comes out is one batch with one path in
// it, and nothing behind it.
func TestOneSaveIsOneReport(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	w, err := filesystem.Watch(t.Context(), domain.Vault{Path: root},
		filesystem.Options{Window: 200 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	for i := range 5 {
		write(t, root, "Note.md", fmt.Sprintf("# Note\n\nrevision %d\n", i))
	}

	select {
	case paths := <-w.Changes:
		if !slices.Equal(paths, []string{"Note.md"}) {
			t.Fatalf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was reported")
	}

	select {
	case again := <-w.Changes:
		t.Errorf("a second batch followed: %v", again)
	case <-time.After(300 * time.Millisecond):
	}
}

// TestAFolderThatGoesAwayCannotBeAnsweredFromDisk. What it held is known to the
// index, so the watcher asks for the vault to be read again rather than
// reporting paths it cannot name.
func TestAFolderThatGoesAwayCannotBeAnsweredFromDisk(t *testing.T) {
	root := vaultOf(t, map[string]string{
		"Note.md":            "# Note\n",
		"projects/Plan.md":   "# Plan\n",
		"projects/deep/A.md": "# A\n",
	}, nil)
	w, err := filesystem.Watch(t.Context(), domain.Vault{Path: root}, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(filepath.Join(root, "projects"), filepath.Join(t.TempDir(), "gone")); err != nil {
		t.Fatal(err)
	}

	select {
	case <-w.Lost:
	case paths := <-w.Changes:
		t.Fatalf("reported %v — the notes under a moved folder cannot be named from disk", paths)
	case <-time.After(5 * time.Second):
		t.Fatal("a folder left the vault and nothing was said")
	}
}

// TestAFolderWithADotInItsNameIsStillAFolder. A name says nothing about what a
// path was: `2026.archive` is a folder and `Note.md.tmp` is not, and only one of
// them takes notes with it.
func TestAFolderWithADotInItsNameIsStillAFolder(t *testing.T) {
	root := vaultOf(t, map[string]string{
		"Note.md":              "# Note\n",
		"2026.archive/Kept.md": "# Kept\n",
	}, nil)
	w, err := filesystem.Watch(t.Context(), domain.Vault{Path: root}, filesystem.Options{})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(filepath.Join(root, "2026.archive")); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-w.Lost:
			return
		case paths := <-w.Changes:
			// Removing the folder removes its notes first, and those it can
			// name. The folder itself it cannot.
			for _, path := range paths {
				if path == "2026.archive" {
					t.Fatalf("reported %v — a folder is not a note", paths)
				}
			}
		case <-deadline:
			t.Fatal("a folder left the vault and nothing was said")
		}
	}
}

// TestFoldingGoesOnWhileNobodyIsListening. Whoever listens takes as long as a
// reindex takes; the operating system does not wait for it, and a fold that
// waited would stop emptying the backlog and lose what came after.
func TestFoldingGoesOnWhileNobodyIsListening(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	w, err := filesystem.Watch(t.Context(), domain.Vault{Path: root},
		filesystem.Options{Window: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}

	// Two changes, well apart, and nothing reading in between.
	write(t, root, "First.md", "# First\n")
	time.Sleep(400 * time.Millisecond)
	write(t, root, "Second.md", "# Second\n")
	time.Sleep(400 * time.Millisecond)

	select {
	case paths := <-w.Changes:
		slices.Sort(paths)
		if !slices.Equal(paths, []string{"First.md", "Second.md"}) {
			t.Fatalf("the first batch was %v — the second change was not folded into it", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was reported")
	}
}

// TestAnEditIsReported.
func TestAnEditIsReported(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	next := watching(t, root)

	write(t, root, "Note.md", "# Note\n\nedited\n")

	if got := next(); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("reported %v", got)
	}
}

// TestASaveThroughATemporaryFileIsOneChange is how editors write: a file beside
// the original, renamed over the top.
func TestASaveThroughATemporaryFileIsOneChange(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	next := watching(t, root)

	temp := filepath.Join(root, "Note.md.tmp")
	if err := os.WriteFile(temp, []byte("# Note\n\nedited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temp, filepath.Join(root, "Note.md")); err != nil {
		t.Fatal(err)
	}

	if got := next(); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("reported %v, want the note once and not the temporary file", got)
	}
}

// TestANewNoteInANewFolderIsReported: the watch is on the tree, and a folder
// made after it started is part of that tree.
func TestANewNoteInANewFolderIsReported(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	next := watching(t, root)

	write(t, root, "later/Added.md", "# Added\n")

	if got := next(); !slices.Contains(got, "later/Added.md") {
		t.Errorf("reported %v", got)
	}
}

// TestWhatTheVaultIgnoresIsNotReported. The rules answer for the watcher and
// the walk alike, so a vault is not indexed differently depending on which one
// found the file.
func TestWhatTheVaultIgnoresIsNotReported(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, []string{"archive/"})
	next := watching(t, root)

	write(t, root, "archive/Old.md", "# Old\n")
	write(t, root, ".#Note.md", "lock\n")
	write(t, root, "Note.md", "# Note\n\nedited\n")

	if got := next(); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("reported %v, want only the note the vault admits to", got)
	}
}
