package run

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

var somewhere = domain.Vault{ID: "somewhere"}

// recorder is a worker whose each hands the names it was given to a channel,
// and holds the first run on a gate.
type recorder struct {
	*Worker
	mu      sync.Mutex
	claimed []string
	reached chan int
	first   chan struct{}
	rest    chan struct{}
}

func newRecorder(t *testing.T, ctx context.Context) *recorder {
	t.Helper()
	r := &recorder{reached: make(chan int, 8), first: make(chan struct{}), rest: make(chan struct{})}
	r.Worker = NewWorker(ctx, NewTasks(), func(_ context.Context, _ domain.Vault, path string) {
		r.mu.Lock()
		r.claimed = append(r.claimed, path)
		n := len(r.claimed)
		r.mu.Unlock()
		r.reached <- n
		if n == 1 {
			<-r.first
		} else {
			<-r.rest
		}
	})
	return r
}

func (r *recorder) countTaken() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.claimed)
}

// instant is a test naming a source at the instant a run finds the line empty.
// The ask is made from a goroutine, because the run is holding the lock the ask
// waits at, and the wait is what puts the two into the same instant.
type instant struct {
	once    sync.Once
	claimed chan windowport.StartOutcome
}

func newInstant() *instant { return &instant{claimed: make(chan windowport.StartOutcome, 1)} }

// nameOnce names the source, once, and leaves the ask standing at the lock
// before the run that called this carries on.
func (c *instant) nameOnce(name func() windowport.StartOutcome) {
	c.once.Do(func() {
		go func() { c.claimed <- name() }()
		time.Sleep(20 * time.Millisecond)
	})
}

func (c *instant) getOutcome(t *testing.T) windowport.StartOutcome {
	t.Helper()
	select {
	case task := <-c.claimed:
		return task
	case <-time.After(10 * time.Second):
		t.Fatal("the ask made as the line emptied was never answered")
		return windowport.Queued
	}
}

func waitForRun(t *testing.T, reached <-chan int) int {
	t.Helper()
	select {
	case n := <-reached:
		return n
	case <-time.After(10 * time.Second):
		t.Fatal("no source was taken")
		return 0
	}
}

// A source still in line when the application closes stays in line, and the run
// that gave up the line stops the running.
func TestASourceInLineOutlivesTheContext(t *testing.T) {
	var ran atomic.Int32
	w := NewWorker(t.Context(), nil, func(context.Context, domain.Vault, string) { ran.Add(1) })

	finished, stop := context.WithCancel(t.Context())
	stop()

	w.Add(somewhere, "a.pdf")
	w.Claim()
	w.DrainAndRelease(finished, nil)

	if w.CountWaiting() != 1 {
		t.Errorf("%d sources are in line after the context ended", w.CountWaiting())
	}
	if ran.Load() != 0 {
		t.Errorf("%d sources ran after the context ended", ran.Load())
	}
	if w.IsRunning() {
		t.Error("a run that gave up the line is still the one running")
	}
}

// A source named at the instant the line empties is run. The run that finds the
// line empty stops the running under the same lock, so the ask that follows is
// told it began and runs the source itself.
//
// The run that ended stops nothing but itself. Its end stands after it let the
// lock go, by which time the run it handed the line to is the one running, and
// a third source named then is told it waits its turn.
func TestASourceNamedAsTheLineEmptiesIsRun(t *testing.T) {
	r := newRecorder(t, t.Context())
	releaseRest := sync.OnceFunc(func() { close(r.rest) })
	defer func() {
		releaseRest()
		r.Wait()
	}()

	crossed := newInstant()
	r.idle = func() { crossed.nameOnce(func() windowport.StartOutcome { return r.Start(somewhere, "b.pdf") }) }

	if got := r.Start(somewhere, "a.pdf"); got != windowport.Began {
		t.Fatalf("the first source was not run: %v", got)
	}
	if n := waitForRun(t, r.reached); n != 1 {
		t.Fatalf("the first source is the %dth", n)
	}
	close(r.first)

	if outcome := crossed.getOutcome(t); outcome != windowport.Began {
		t.Fatalf("the source named as the line emptied received %v", outcome)
	}
	if n := waitForRun(t, r.reached); n != 2 {
		t.Fatalf("the source named as the line emptied is the %dth", n)
	}

	r.Release(1)
	if got := r.Start(somewhere, "c.pdf"); got != windowport.Queued {
		t.Errorf("a source named while one was running received %v", got)
	}

	releaseRest()
	r.Wait()

	if r.countTaken() != 3 {
		t.Errorf("%d sources ran", r.countTaken())
	}
	if r.CountWaiting() != 0 {
		t.Errorf("%d sources were left in line", r.CountWaiting())
	}
}

// Drain leaves the running as it is, so a round can drain between the sources
// it chooses itself.
func TestDrainDoesNotStopTheRunning(t *testing.T) {
	w := NewWorker(t.Context(), nil, func(context.Context, domain.Vault, string) {})
	w.Add(somewhere, "a.pdf")
	w.Claim()
	w.Drain(t.Context())

	if !w.IsRunning() {
		t.Error("a drain stopped the running")
	}
	if w.CountWaiting() != 0 {
		t.Errorf("%d sources were left in line", w.CountWaiting())
	}
}

// A claim fails while a run holds the line.
func TestAClaimIsRefusedWhileARunContainsTheLine(t *testing.T) {
	w := NewWorker(t.Context(), nil, func(context.Context, domain.Vault, string) {})
	if _, free := w.Claim(); !free {
		t.Fatal("an idle worker rejected a claim")
	}
	if _, free := w.Claim(); free {
		t.Error("a second claim was granted")
	}
}

// A source named twice waits in line once.
func TestASourceNamedTwiceWaitsOnce(t *testing.T) {
	w := NewWorker(t.Context(), nil, func(context.Context, domain.Vault, string) {})
	if !w.Add(somewhere, "a.pdf") || w.Add(somewhere, "a.pdf") {
		t.Error("a source named twice went into the line twice")
	}
	if w.CountWaiting() != 1 {
		t.Errorf("%d sources are in line", w.CountWaiting())
	}
}

// Rounds are run until the context ends.
func TestRoundsRepeatUntilTheContextEnds(t *testing.T) {
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	w := NewWorker(ctx, nil, nil)
	var rounds atomic.Int32
	w.Go(func() {
		w.RepeatRounds(ctx, func(ctx context.Context, _ time.Duration) error { return ctx.Err() }, time.Millisecond, func() {
			if rounds.Add(1) == 3 {
				stop()
			}
		})
	})
	w.Wait()
	if rounds.Load() != 3 {
		t.Errorf("%d rounds ran", rounds.Load())
	}
}
