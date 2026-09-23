package source

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// whenIdle is what a run calls where it has found the line empty and is about
// to stop the running. It stands beside the running and is set under the lock
// the running is kept under.
func (r *RecognitionWorker) whenIdle(idle func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.idle = idle
}

// instant is a test naming a source at the instant a run finds the line empty.
// The ask is made from a goroutine, because the run is holding the lock the ask
// waits at, and the wait is what puts the two into the same instant.
type instant struct {
	once  sync.Once
	taken chan port.StartOutcome
}

func newInstant() *instant { return &instant{taken: make(chan port.StartOutcome, 1)} }

// nameOnce names the source, once, and leaves the ask standing at the lock
// before the run that called this carries on.
func (c *instant) nameOnce(name func() port.StartOutcome) {
	c.once.Do(func() {
		go func() { c.taken <- name() }()
		time.Sleep(20 * time.Millisecond)
	})
}

// getOutcome is what the source named at that instant was told.
func (c *instant) getOutcome(t *testing.T) port.StartOutcome {
	t.Helper()
	select {
	case taken := <-c.taken:
		return taken
	case <-time.After(10 * time.Second):
		t.Fatal("the ask made as the line emptied was never answered")
		return port.Queued
	}
}

// waitForReading waits for the next reading to reach the models, and says which
// reading it is.
func waitForReading(t *testing.T, reading <-chan int) int {
	t.Helper()
	select {
	case n := <-reading:
		return n
	case <-time.After(10 * time.Second):
		t.Fatal("no reading reached the models")
		return 0
	}
}

// A document still in line when the application closes is left in line. A run
// that took it and then dropped it would leave a document never read and never
// reported, on a count that fell as though it had been.
func TestADocumentInLineOutlivesTheContext(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	over, stop := context.WithCancel(t.Context())
	stop()

	w.mu.Lock()
	w.queue.add(somewhere, "a.pdf")
	w.isRunning = true
	w.mu.Unlock()
	w.drain(over)

	if w.CountWaiting() != 1 {
		t.Errorf("%d documents are in line after the context ended", w.CountWaiting())
	}
	if w.countOpens() != 0 {
		t.Errorf("a recogniser was opened %d times after the context ended", w.countOpens())
	}
	// The run that gave up the line stops the running, so nothing is left
	// standing over a line it will never come back to.
	if w.IsRunning() {
		t.Error("a run that gave up the line is still the one running")
	}
}

// A document named at the instant the line empties is read. The run that finds
// the line empty stops the running under the same lock, so the ask that follows
// is told it began and reads the document itself.
//
// Nothing else reads this line.
//
// The run that ended stops nothing but itself. Its end stands after it let the
// lock go, by which time the run it handed the line to is the one running, and
// a third document named then is told it waits its turn.
func TestADocumentNamedAsTheLineEmptiesIsRead(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	// The first reading is held until this test lets it end, and every reading
	// after it until this test is over.
	first, rest := make(chan struct{}), make(chan struct{})
	releaseRest := sync.OnceFunc(func() { close(rest) })
	defer func() {
		releaseRest()
		w.Wait()
	}()

	reading := make(chan int, 8)
	w.RecognitionWorker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		w.mu.Lock()
		w.open++
		n := w.open
		w.mu.Unlock()
		reading <- n
		if n == 1 {
			<-first
		} else {
			<-rest
		}
		return nil, nil, errors.New("no models on this machine")
	}

	crossed := newInstant()
	w.whenIdle(func() { crossed.nameOnce(func() port.StartOutcome { return w.Start(somewhere, "b.pdf") }) })

	if got := w.Start(somewhere, "a.pdf"); got != port.Began {
		t.Fatalf("the first document was not read: %v", got)
	}
	if n := waitForReading(t, reading); n != 1 {
		t.Fatalf("the first reading is the %dth", n)
	}
	close(first)

	if taken := crossed.getOutcome(t); taken != port.Began {
		t.Fatalf("the document named as the line emptied was told %v", taken)
	}
	if n := waitForReading(t, reading); n != 2 {
		t.Fatalf("the document named as the line emptied is the %dth reading", n)
	}

	// The run that emptied the line ends now. Its end stands anywhere after it
	// let the lock go, and it stops nothing but itself: the run it handed the
	// line to is the one running, and a third document waits its turn.
	w.RecognitionWorker.stopRun(1)
	if got := w.Start(somewhere, "c.pdf"); got != port.Queued {
		t.Errorf("a document named while one was being read was told %v", got)
	}

	releaseRest()
	w.waitUntilDone(t)
	w.Wait()

	if w.countOpens() != 3 {
		t.Errorf("a recogniser was opened %d times", w.countOpens())
	}
	if w.CountWaiting() != 0 {
		t.Errorf("%d documents were left in line", w.CountWaiting())
	}
}
