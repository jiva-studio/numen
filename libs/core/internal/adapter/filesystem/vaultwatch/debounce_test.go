package vaultwatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/rjeczalik/notify"
)

// event is one change, as delivered by the operating system.
type event struct{ path string }

func (e event) Event() notify.Event { return notify.Write }
func (e event) Path() string        { return e.path }
func (e event) Sys() any            { return nil }

// makeVaultOfNotes makes a vault of notes and the reader that answers for it. The root
// it gives back is the reader's own, which is every link resolved: that is the
// path the operating system names a change at.
func makeVaultOfNotes(t *testing.T, notes int) (root string, reader *vaultreader.VaultReader) {
	t.Helper()
	pos := t.TempDir()
	for i := range notes {
		name := filepath.Join(pos, fmt.Sprintf("Note%d.md", i))
		if err := os.WriteFile(name, []byte("# Note\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reader, err := vaultreader.Open(pos, vaultconfig.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return reader.Root(), reader
}

// shapeOf is a reader's vault currently before the first event.
func shapeOf(t *testing.T, reader *vaultreader.VaultReader) *vaultreader.Folders {
	t.Helper()
	shape, err := vaultreader.ReadShape(reader)
	if err != nil {
		t.Fatal(err)
	}
	return shape
}

// feed sends a burst of notes into the watcher's channel.
func feed(t *testing.T, raw chan notify.EventInfo, root string, from, notes int) {
	t.Helper()
	for i := from; i < from+notes; i++ {
		raw <- event{path: filepath.Join(root, fmt.Sprintf("Note%d.md", i))}
	}
}

// waitForQueue waits until the queue holds what the burst produced, so the input to
// debouncing is determined by the burst and not by timing.
func waitForQueue(t *testing.T, q *queue, is func() bool) {
	t.Helper()
	deadline := testsupport.Machine.Now().Add(5 * time.Second)
	for {
		q.mu.Lock()
		isDone := is()
		q.mu.Unlock()
		if isDone {
			return
		}
		if testsupport.Machine.Now().After(deadline) {
			t.Fatal("the burst never reached the queue")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestABurstPastTheBacklogMeansTheVaultIsReadAgain. The burst is over before
// the debouncing looks at anything, so the answer does not depend on how fast the
// next event arrives — or on whether one arrives at all.
func TestABurstPastTheBacklogMeansTheVaultIsReadAgain(t *testing.T) {
	root, reader := makeVaultOfNotes(t, 5)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	// One event past the bound, and everything held goes with it.
	raw := make(chan notify.EventInfo, 4)
	queue := newQueue(4)
	go forwardEvents(ctx, raw, queue)
	feed(t, raw, root, 0, 5)
	waitForQueue(t, queue, func() bool { return queue.isOver })

	changes := make(chan []string)
	missing := make(chan struct{}, 1)
	go debounce(ctx, shapeOf(t, reader), vaultconfig.Options{DebounceDelay: 10 * time.Millisecond}, queue, changes, missing)

	select {
	case <-missing:
	case paths := <-changes:
		t.Fatalf("reported %v — what a burst past the backlog dropped cannot be named", paths)
	case <-time.After(5 * time.Second):
		t.Fatal("a burst past the backlog was not reported as lost")
	}
}

// TestABurstInsideTheBacklogIsNotALoss. Nothing was dropped, so no state is unknown,
// and a walk of the whole vault would be unnecessary work.
func TestABurstInsideTheBacklogIsNotALoss(t *testing.T) {
	root, reader := makeVaultOfNotes(t, 4)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	raw := make(chan notify.EventInfo, 4)
	queue := newQueue(4)
	go forwardEvents(ctx, raw, queue)
	feed(t, raw, root, 0, 4)
	waitForQueue(t, queue, func() bool { return len(queue.events) == 4 })

	changes := make(chan []string)
	missing := make(chan struct{}, 1)
	go debounce(ctx, shapeOf(t, reader), vaultconfig.Options{DebounceDelay: 10 * time.Millisecond}, queue, changes, missing)

	select {
	case <-missing:
		t.Fatal("a backlog that was never past its bound was read again")
	case paths := <-changes:
		slices.Sort(paths)
		want := []string{"Note0.md", "Note1.md", "Note2.md", "Note3.md"}
		if !slices.Equal(paths, want) {
			t.Fatalf("reported %v, want %v", paths, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a burst inside the backlog was never reported")
	}
}

// TestAListenerThatDoesNotTakeDoesNotStopTheReading. A listener takes as long as a
// reindex takes, and the operating system does not wait. Events that arrive while a
// batch is undelivered are read, debounced and reported after it.
func TestAListenerThatDoesNotTakeDoesNotStopTheReading(t *testing.T) {
	root, reader := makeVaultOfNotes(t, 9)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	raw := make(chan notify.EventInfo, 4)
	queue := newQueue(64)
	go forwardEvents(ctx, raw, queue)

	changes := make(chan []string)
	missing := make(chan struct{}, 1)
	go debounce(ctx, shapeOf(t, reader), vaultconfig.Options{DebounceDelay: 10 * time.Millisecond}, queue, changes, missing)

	// One note, debounced and held, leaves the goroutine with a batch that no listener
	// takes.
	feed(t, raw, root, 0, 1)
	time.Sleep(200 * time.Millisecond)

	// Eight more, through a channel with capacity four, while that batch is still
	// undelivered.
	feed(t, raw, root, 1, 8)

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-missing:
			t.Fatal("nothing was dropped and the whole vault was read again")
		case paths := <-changes:
			if slices.Contains(paths, "Note8.md") {
				return
			}
		case <-deadline:
			t.Fatal("what arrived behind an undelivered batch was never reported")
		}
	}
}
