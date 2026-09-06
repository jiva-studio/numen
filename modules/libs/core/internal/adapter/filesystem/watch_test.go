package filesystem_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// watching follows a vault and gives back a function that waits for the next
// batch of changed paths.
func watching(t *testing.T, root string) func() []string {
	t.Helper()
	changes, _, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}
	return func() []string {
		select {
		case paths := <-changes:
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
	changes, _, err := filesystem.Watcher{Options: filesystem.Options{Hold: 200 * time.Millisecond}}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	for i := range 5 {
		write(t, root, "Note.md", fmt.Sprintf("# Note\n\nrevision %d\n", i))
	}

	select {
	case paths := <-changes:
		if !slices.Equal(paths, []string{"Note.md"}) {
			t.Fatalf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was reported")
	}

	select {
	case again := <-changes:
		t.Errorf("a second batch followed: %v", again)
	case <-time.After(300 * time.Millisecond):
	}
}

// TestAFolderThatGoesAwayCannotBeAnsweredFromDisk. What it held is known to the
// index, so the watcher asks for the vault to be read again.
func TestAFolderThatGoesAwayCannotBeAnsweredFromDisk(t *testing.T) {
	root := vaultOf(t, map[string]string{
		"Note.md":            "# Note\n",
		"projects/Plan.md":   "# Plan\n",
		"projects/deep/A.md": "# A\n",
	}, nil)
	changes, lost, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(filepath.Join(root, "projects"), filepath.Join(t.TempDir(), "gone")); err != nil {
		t.Fatal(err)
	}

	select {
	case <-lost:
	case paths := <-changes:
		t.Fatalf("reported %v — the notes under a moved folder cannot be named from disk", paths)
	case <-time.After(5 * time.Second):
		t.Fatal("a folder left the vault and nothing was said")
	}
}

// TestAFolderAlreadyThereIsNotItsWholeContents. A folder is named in its own
// right when something inside it changes, and that something is named too. The
// folder is known, so it adds nothing to what its own contents already said.
func TestAFolderAlreadyThereIsNotItsWholeContents(t *testing.T) {
	root := vaultOf(t, map[string]string{
		"Note.md":      "# Note\n",
		"Untouched.md": "# Untouched\n",
		"Aside.md":     "# Aside\n",
	}, nil)
	// A watch opened on a folder written a moment ago is told what was already
	// in it, and one of those cannot be told from a file that has just arrived.
	time.Sleep(500 * time.Millisecond)
	next := watching(t, root)

	now := time.Now()
	if err := os.Chtimes(root, now, now); err != nil {
		t.Fatal(err)
	}
	write(t, root, "Note.md", "# Note\n\nedited\n")

	if got := next(); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("reported %v, want only the note that changed", got)
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
	changes, lost, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(filepath.Join(root, "2026.archive")); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-lost:
			return
		case paths := <-changes:
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

// TestDebouncingGoesOnWhileNobodyIsListening. Whoever listens takes as long as
// a reindex takes; the operating system does not wait for it, and a debounce that
// waited would stop emptying the backlog and lose what came after.
func TestDebouncingGoesOnWhileNobodyIsListening(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	changes, _, err := filesystem.Watcher{Options: filesystem.Options{Hold: 50 * time.Millisecond}}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	// Two changes, well apart, and nothing reading in between.
	write(t, root, "First.md", "# First\n")
	time.Sleep(400 * time.Millisecond)
	write(t, root, "Second.md", "# Second\n")
	time.Sleep(400 * time.Millisecond)

	select {
	case paths := <-changes:
		// Both changes stand in the one batch, and each path stands in it once.
		// A batch may name more than was changed: a watch just opened on a vault
		// is told about files that were already there, and one of those cannot
		// be told from a file that has just arrived.
		for _, want := range []string{"First.md", "Second.md"} {
			if !slices.Contains(paths, want) {
				t.Fatalf("the first batch was %v — %s was not debounced into it", paths, want)
			}
		}
		sorted := slices.Clone(paths)
		slices.Sort(sorted)
		if len(slices.Compact(sorted)) != len(paths) {
			t.Fatalf("the batch was %v — a path was named twice", paths)
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

// TestABookAppearingIsReported. The watcher filters through the same answer the
// walk does, so dropping a book into a folder is the whole gesture.
func TestABookAppearingIsReported(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	changes, _, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.WriteBook(t, root, "library/Dropped.epub")

	// A new folder and the file inside it arrive as their own events, and which
	// batch carries the book depends on which of them the walk of the folder
	// caught.
	deadline := time.After(5 * time.Second)
	for {
		select {
		case paths := <-changes:
			if slices.Contains(paths, "library/Dropped.epub") {
				return
			}
		case <-deadline:
			t.Fatal("a book appeared in the vault and nothing was said")
		}
	}
}

// TestAnImageAppearingIsNotReported. Classification is by type on both paths:
// what the walk does not report the watcher does not report either.
func TestAnImageAppearingIsNotReported(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	next := watching(t, root)

	write(t, root, "assets/scan.png", "PNG\n")
	write(t, root, "Note.md", "# Note\n\nedited\n")

	if got := next(); !slices.Equal(got, []string{"Note.md"}) {
		t.Errorf("reported %v, want only the note", got)
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

// A file is followed by its folder, so it is still followed after it is
// replaced: a database written beside itself and renamed over is the same file
// to whoever reads it.
func TestAFileIsFollowedThroughBeingReplaced(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, "index.db")
	if err := os.WriteFile(at, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := filesystem.Watcher{}.File(t.Context(), at)
	if err != nil {
		t.Fatal(err)
	}
	waits := func(why string) {
		t.Helper()
		select {
		case <-moved:
		case <-time.After(5 * time.Second):
			t.Fatal(why)
		}
	}

	// The journal beside it is the same file changing.
	if err := os.WriteFile(at+"-wal", []byte("written"), 0o644); err != nil {
		t.Fatal(err)
	}
	waits("the journal was written and nothing was reported")

	beside := filepath.Join(dir, "index.db.new")
	if err := os.WriteFile(beside, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(beside, at); err != nil {
		t.Fatal(err)
	}
	waits("the file was replaced and nothing was reported")

	// A file of another name in the same folder is not this file. What the
	// replacing left waiting is taken first: one message stands for whatever
	// happened before it was read.
	for draining := true; draining; {
		select {
		case <-moved:
		case <-time.After(200 * time.Millisecond):
			draining = false
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "other"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-moved:
		t.Error("another file in the folder was reported as this one")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestAFolderTheVaultIgnoresGoingAwayIsNotALoss. `git gc`, a fetch and a
// checkout each take a folder inside `.git` away, and an editor takes its own
// scratch folder away on every save. The vault holds nothing under any of them,
// so nothing about it became unknowable.
func TestAFolderTheVaultIgnoresGoingAwayIsNotALoss(t *testing.T) {
	root := vaultOf(t, map[string]string{
		"Note.md":                "# Note\n",
		".git/objects/tmp_abc":   "pack\n",
		"node_modules/left/A.md": "# A\n",
		"archive/deep/Buried.md": "# Buried\n",
	}, []string{"node_modules/", "archive/"})
	changes, lost, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	for _, gone := range []string{".git", "node_modules", "archive"} {
		if err := os.RemoveAll(filepath.Join(root, gone)); err != nil {
			t.Fatal(err)
		}
	}
	write(t, root, "Note.md", "# Note\n\nedited\n")

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-lost:
			t.Fatal("a folder the vault ignores went away and the whole vault was read again")
		case paths := <-changes:
			if !slices.Equal(paths, []string{"Note.md"}) {
				t.Fatalf("reported %v, want only the note the vault admits to", paths)
			}
			return
		case <-deadline:
			t.Fatal("the note was written and nothing was said")
		}
	}
}

// TestAFolderTheVaultIgnoresArrivingIsNotRemembered. A checkout puts a tree
// inside `.git` and the next one takes it away again. The watcher stops at the
// folder the walk stops at, so neither half of that is the vault's.
func TestAFolderTheVaultIgnoresArrivingIsNotRemembered(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	changes, lost, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	// Events arrive in the order they happened, so a batch naming the note is
	// the watcher having already seen the folder arrive.
	write(t, root, ".git/refs/heads/main.md", "# Not a note\n")
	write(t, root, "Note.md", "# Note\n\nedited\n")
	select {
	case <-changes:
	case <-lost:
		t.Fatal("a folder the vault ignores arrived and the whole vault was read again")
	case <-time.After(5 * time.Second):
		t.Fatal("the note was written and nothing was said")
	}

	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "Note.md", "# Note\n\nedited again\n")

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-lost:
			t.Fatal("a folder the vault ignores came and went, and the whole vault was read again")
		case paths := <-changes:
			if !slices.Equal(paths, []string{"Note.md"}) {
				t.Fatalf("reported %v, want only the note", paths)
			}
			return
		case <-deadline:
			t.Fatal("the note was written and nothing was said")
		}
	}
}

// TestAVaultReachedThroughALinkIsFollowedLikeAnyOther. The operating system
// names the file it saw change, and what it names is the path with every link
// resolved. A vault in a synced folder is reached through one.
func TestAVaultReachedThroughALinkIsFollowedLikeAnyOther(t *testing.T) {
	physical := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	link := filepath.Join(t.TempDir(), "vault")
	if err := os.Symlink(physical, link); err != nil {
		t.Fatal(err)
	}

	changes, lost, err := filesystem.Watcher{}.Watch(t.Context(), domain.Vault{Path: link})
	if err != nil {
		t.Fatal(err)
	}
	write(t, physical, "Note.md", "# Note\n\nedited\n")

	select {
	case paths := <-changes:
		if !slices.Equal(paths, []string{"Note.md"}) {
			t.Errorf("reported %v, want the note it named", paths)
		}
	case <-lost:
		t.Fatal("the whole vault was read again — the path the system named was taken for one outside it")
	case <-time.After(5 * time.Second):
		t.Fatal("the note was written and nothing was said")
	}
}

// TestAVaultThatIsNeverStillIsStillReported. The hold runs from the first event
// of a batch. A vault written to without pause — a sync client, a checkout —
// never stops long enough for a hold that begins again at every event.
func TestAVaultThatIsNeverStillIsStillReported(t *testing.T) {
	root := vaultOf(t, map[string]string{"Note.md": "# Note\n"}, nil)
	changes, _, err := filesystem.Watcher{
		Options: filesystem.Options{Hold: 200 * time.Millisecond},
	}.Watch(t.Context(), domain.Vault{Path: root})
	if err != nil {
		t.Fatal(err)
	}

	stop := make(chan struct{})
	writing := make(chan struct{})
	t.Cleanup(func() { close(stop); <-writing })
	go func() {
		defer close(writing)
		for i := range 100 {
			name := filepath.Join(root, fmt.Sprintf("Note%d.md", i))
			if err := os.WriteFile(name, []byte("# Note\n"), 0o644); err != nil {
				return
			}
			select {
			case <-stop:
				return
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()

	select {
	case <-changes:
	case <-writing:
		t.Fatal("a vault written to without pause was never reported")
	}
}
