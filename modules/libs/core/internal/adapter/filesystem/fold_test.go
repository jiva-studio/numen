package filesystem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/rjeczalik/notify"
)

// event is one change, as the operating system hands it over.
type event struct{ path string }

func (e event) Event() notify.Event { return notify.Write }
func (e event) Path() string        { return e.path }
func (e event) Sys() any            { return nil }

// watched makes a vault of notes and the reader that answers for it. The root
// it gives back is the reader's own, which is every link resolved: that is the
// path the operating system names a change at.
func watched(t *testing.T, notes int) (root string, reader *VaultReader) {
	t.Helper()
	at := t.TempDir()
	for i := range notes {
		name := filepath.Join(at, fmt.Sprintf("Note%d.md", i))
		if err := os.WriteFile(name, []byte("# Note\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reader, err := Open(at, Options{})
	if err != nil {
		t.Fatal(err)
	}
	return reader.Root(), reader
}

// shapeOf is a reader's vault as it stands before the first event.
func shapeOf(t *testing.T, reader *VaultReader) *folders {
	t.Helper()
	shape, err := remembered(reader)
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

// settles waits for the queue to be what the burst made it, so that what the
// folder is started on is decided by the burst and not by when it started.
func settles(t *testing.T, q *queue, is func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		q.mu.Lock()
		done := is()
		q.mu.Unlock()
		if done {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the burst never reached the queue")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestABurstPastTheBacklogMeansTheVaultIsReadAgain. The burst is over before
// the folder looks at anything, so the answer does not depend on how fast the
// next event arrives — or on whether one arrives at all.
func TestABurstPastTheBacklogMeansTheVaultIsReadAgain(t *testing.T) {
	root, reader := watched(t, 5)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	// One event past the bound, and everything held goes with it.
	raw := make(chan notify.EventInfo, 4)
	waiting := newQueue(4)
	go drain(ctx, raw, waiting)
	feed(t, raw, root, 0, 5)
	settles(t, waiting, func() bool { return waiting.over })

	changes := make(chan []string)
	lost := make(chan struct{}, 1)
	go fold(ctx, shapeOf(t, reader), Options{Hold: 10 * time.Millisecond}, waiting, changes, lost)

	select {
	case <-lost:
	case paths := <-changes:
		t.Fatalf("reported %v — what a burst past the backlog dropped cannot be named", paths)
	case <-time.After(5 * time.Second):
		t.Fatal("a burst past the backlog was not reported as lost")
	}
}

// TestABurstInsideTheBacklogIsNotALoss. Nothing was dropped, so nothing became
// unknowable, and a walk of the whole vault is work nobody asked for.
func TestABurstInsideTheBacklogIsNotALoss(t *testing.T) {
	root, reader := watched(t, 4)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	raw := make(chan notify.EventInfo, 4)
	waiting := newQueue(4)
	go drain(ctx, raw, waiting)
	feed(t, raw, root, 0, 4)
	settles(t, waiting, func() bool { return len(waiting.events) == 4 })

	changes := make(chan []string)
	lost := make(chan struct{}, 1)
	go fold(ctx, shapeOf(t, reader), Options{Hold: 10 * time.Millisecond}, waiting, changes, lost)

	select {
	case <-lost:
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

// TestAListenerThatDoesNotTakeDoesNotStopTheReading. Whoever listens takes as
// long as a reindex takes, and the operating system does not wait for it. What
// arrives while a batch stands undelivered is read, folded and reported after
// it.
func TestAListenerThatDoesNotTakeDoesNotStopTheReading(t *testing.T) {
	root, reader := watched(t, 9)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	raw := make(chan notify.EventInfo, 4)
	waiting := newQueue(64)
	go drain(ctx, raw, waiting)

	changes := make(chan []string)
	lost := make(chan struct{}, 1)
	go fold(ctx, shapeOf(t, reader), Options{Hold: 10 * time.Millisecond}, waiting, changes, lost)

	// One note, folded and held, leaves the folder with a batch nobody is
	// taking.
	feed(t, raw, root, 0, 1)
	time.Sleep(200 * time.Millisecond)

	// Eight more, through a channel that holds four, while that batch still
	// stands.
	feed(t, raw, root, 1, 8)

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-lost:
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

// TestTheQueueDropsAtItsBoundAndSaysSo. A bound with an honest overflow is what
// keeps a checkout of a large repository from becoming memory the machine does
// not have.
func TestTheQueueDropsAtItsBoundAndSaysSo(t *testing.T) {
	waiting := newQueue(2)
	for i := range 5 {
		waiting.put(event{path: fmt.Sprintf("Note%d.md", i)})
	}

	events, over, done := waiting.take()
	if !over {
		t.Fatal("the queue was filled past its bound and did not say so")
	}
	if done {
		t.Error("the queue said the watcher had stopped")
	}
	if len(events) > 2 {
		t.Errorf("the queue held %d events, past its bound of 2", len(events))
	}

	if _, over, _ := waiting.take(); over {
		t.Error("one overflow was reported twice")
	}
}
