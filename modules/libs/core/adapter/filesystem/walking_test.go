package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
)

// shaped is a vault holding one note, and the shape of it taken before anything
// else arrives.
func shaped(t *testing.T) (string, *folders) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Note.md"), []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reader, err := Open(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	shape, err := remembered(reader)
	if err != nil {
		t.Fatal(err)
	}
	return root, shape
}

// filled writes a folder of notes into the vault, the way a sync client
// unpacking an archive does.
func filled(t *testing.T, root, name string, notes int) string {
	t.Helper()
	at := filepath.Join(root, name)
	if err := os.MkdirAll(at, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range notes {
		one := filepath.Join(at, "Note"+string(rune('a'+i%26))+string(rune('a'+i/26))+".md")
		if err := os.WriteFile(one, []byte("# One\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return at
}

// TestAFolderNewToTheWatchIsHandedOverRatherThanWalked. The one goroutine that
// empties the backlog walks nothing: a folder unpacked into the vault would
// hold it for the length of the walk, the operating system would overflow the
// backlog meanwhile, and the answer to an overflow is reading the whole vault
// again — which is the rescan this watch exists to avoid.
func TestAFolderNewToTheWatchIsHandedOverRatherThanWalked(t *testing.T) {
	root, shape := shaped(t)
	at := filled(t, root, "library", 3)

	paths, whole, walk := shape.concerns(at)
	if whole {
		t.Fatal("a folder that arrived was answered as the whole vault")
	}
	if len(paths) != 0 {
		t.Errorf("the folder was walked where the events are drained: %v", paths)
	}
	if walk != at {
		t.Fatalf("the folder was not handed over to be walked: %q", walk)
	}

	held, whole := shape.inside(t.Context(), walk)
	if whole {
		t.Fatal("a folder of three notes was answered as the whole vault")
	}
	if len(held) != 3 {
		t.Errorf("the walk of the folder found %v", held)
	}
}

// TestAWalkStopsWhenTheWatchDoes. A vault being let go of is not a vault to go
// on reading the disk for.
func TestAWalkStopsWhenTheWatchDoes(t *testing.T) {
	root, shape := shaped(t)
	at := filled(t, root, "library", 3)

	ctx, stop := context.WithCancel(t.Context())
	stop()
	if held, whole := shape.inside(ctx, at); whole || len(held) != 0 {
		t.Errorf("a walk under a context that is done found %v (whole %v)", held, whole)
	}
}

// TestAFolderThatArrivesIsReportedThroughTheFold. The walk stands beside the
// fold, so what it finds has to come back to it.
func TestAFolderThatArrivesIsReportedThroughTheFold(t *testing.T) {
	root, shape := shaped(t)
	at := filled(t, root, "library", 2)

	raw := make(chan notify.EventInfo, 4)
	changes := make(chan []string)
	lost := make(chan struct{}, 1)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	waiting := newQueue(64)
	go drain(ctx, raw, waiting)
	go fold(ctx, shape, Options{Hold: 10 * time.Millisecond}, waiting, changes, lost)

	raw <- event{path: at}

	select {
	case told := <-changes:
		slices.Sort(told)
		if len(told) != 2 {
			t.Fatalf("the folder that arrived was reported as %v", told)
		}
	case <-lost:
		t.Fatal("a folder of two notes was answered as the whole vault")
	case <-time.After(10 * time.Second):
		t.Fatal("the folder that arrived was never reported")
	}
}
