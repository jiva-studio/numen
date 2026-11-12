package vaultwatch

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/rjeczalik/notify"
)

func makeVaultAndShape(t *testing.T) (string, *vaultreader.Folders) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Note.md"), []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reader, err := vaultreader.Open(root, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	shape, err := vaultreader.ReadShape(reader)
	if err != nil {
		t.Fatal(err)
	}
	return root, shape
}

func writeFolderOfNotes(t *testing.T, root, name string, notes int) string {
	t.Helper()
	pos := filepath.Join(root, name)
	if err := os.MkdirAll(pos, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range notes {
		one := filepath.Join(pos, "Note"+string(rune('a'+i%26))+string(rune('a'+i/26))+".md")
		if err := os.WriteFile(one, []byte("# One\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pos
}

// TestAFolderNewToTheWatchIsHandedOverRatherThanWalked. The one goroutine that
// empties the backlog walks nothing: a folder new to the watch is returned
// as a name, and it is walked beside the backlog that goes on emptying.
func TestAFolderNewToTheWatchIsHandedOverInsteadOfWalked(t *testing.T) {
	root, shape := makeVaultAndShape(t)
	pos := writeFolderOfNotes(t, root, "library", 3)

	paths, whole, walk := shape.GetConcernedPaths(pos)
	if whole {
		t.Fatal("a folder that arrived was answered as the whole vault")
	}
	if len(paths) != 0 {
		t.Errorf("the folder was walked where the events are drained: %v", paths)
	}
	if walk != pos {
		t.Fatalf("the folder was not passed to be walked: %q", walk)
	}

	target, whole := shape.GetPathsUnder(t.Context(), walk)
	if whole {
		t.Fatal("a folder of three notes was answered as the whole vault")
	}
	if len(target) != 3 {
		t.Errorf("the walk of the folder found %v", target)
	}
}

// TestAWalkStopsWhenTheWatchDoes. A vault being release is not a vault to go
// on reading the disk for.
func TestAWalkStopsWhenTheWatchDoes(t *testing.T) {
	root, shape := makeVaultAndShape(t)
	pos := writeFolderOfNotes(t, root, "library", 3)

	ctx, stop := context.WithCancel(t.Context())
	stop()
	if target, whole := shape.GetPathsUnder(ctx, pos); whole || len(target) != 0 {
		t.Errorf("a walk under a context that is done found %v (whole %v)", target, whole)
	}
}

// TestAFolderThatArrivesIsReportedThroughTheDebounce. The walk runs alongside
// the debounce, so what it finds has to be reported back through it.
func TestAFolderThatArrivesIsReportedThroughTheDebounce(t *testing.T) {
	root, shape := makeVaultAndShape(t)
	pos := writeFolderOfNotes(t, root, "library", 2)

	raw := make(chan notify.EventInfo, 4)
	changes := make(chan []string)
	missing := make(chan struct{}, 1)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	queue := newQueue(64)
	go forwardEvents(ctx, raw, queue)
	go debounce(ctx, shape, vaultconfig.Options{DebounceDelay: 10 * time.Millisecond}, queue, changes, missing)

	raw <- event{path: pos}

	select {
	case change := <-changes:
		slices.Sort(change)
		if len(change) != 2 {
			t.Fatalf("the folder that arrived was reported as %v", change)
		}
	case <-missing:
		t.Fatal("a folder of two notes was answered as the whole vault")
	case <-time.After(10 * time.Second):
		t.Fatal("the folder that arrived was never reported")
	}
}

// TestAFolderWithNoFileIsAChange. A folder holding no file, or only folders,
// adds no source, yet it is drawn in the tree, so the vault is read again.
func TestAFolderWithNoFileIsAChange(t *testing.T) {
	for name, build := range map[string]func(string) string{
		"empty folder": func(root string) string {
			pos := filepath.Join(root, "Saltworks")
			if err := os.Mkdir(pos, 0o755); err != nil {
				t.Fatal(err)
			}
			return pos
		},
		"folder holding an empty folder": func(root string) string {
			pos := filepath.Join(root, "Saltworks")
			if err := os.MkdirAll(filepath.Join(pos, "Brine tanks"), 0o755); err != nil {
				t.Fatal(err)
			}
			return pos
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, shape := makeVaultAndShape(t)
			pos := build(root)

			raw := make(chan notify.EventInfo, 4)
			changes := make(chan []string, 1)
			missing := make(chan struct{}, 1)
			ctx, stop := context.WithCancel(t.Context())
			defer stop()
			queue := newQueue(64)
			go forwardEvents(ctx, raw, queue)
			go debounce(ctx, shape, vaultconfig.Options{DebounceDelay: 10 * time.Millisecond}, queue, changes, missing)

			raw <- event{path: pos}

			select {
			case <-missing:
			case change := <-changes:
				t.Fatalf("the folder was reported as the sources %v", change)
			case <-time.After(10 * time.Second):
				t.Fatal("the folder that arrived was never reported")
			}
		})
	}
}
