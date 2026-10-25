package run

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// Worker runs the sources a person named one at a time, in the order they named
// them, behind whoever asked.
//
// Nothing is done inside the question that asked for it. The ask starts the
// work and says so, and how far it has got is put in the list of what is being
// done.
type Worker struct {
	tasks *Tasks
	each  func(ctx context.Context, v domain.Vault, path string)

	// ctx is the root context for runs, and going tracks active runs.
	ctx   context.Context
	going sync.WaitGroup

	mu        sync.Mutex
	isRunning bool
	// runs counts the runs that have begun. A run stops the running only while
	// it is still the one running.
	runs  uint64
	queue queue
	// idle is called where a run has found the line empty and is about to stop
	// the running, under the lock both it and the running are held by.
	idle func()
}

// NewWorker is a worker running each source it takes from the line through
// each, reporting into tasks when there are any.
func NewWorker(
	ctx context.Context,
	tasks *Tasks,
	each func(ctx context.Context, v domain.Vault, path string),
) *Worker {
	return &Worker{tasks: tasks, each: each, ctx: ctx}
}

// Context returns the worker's root context.
func (w *Worker) Context() context.Context {
	if w.ctx == nil {
		return context.Background()
	}
	return w.ctx
}

// Wait is every run and every goroutine started through Go, ended.
func (w *Worker) Wait() { w.going.Wait() }

// Go runs do behind the caller, counted by Wait.
func (w *Worker) Go(do func()) {
	w.going.Add(1)
	go func() {
		defer w.going.Done()
		do()
	}()
}

// IsRunning says whether a run holds the line.
func (w *Worker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isRunning
}

// Add puts a source at the back of the line and says whether it went in.
func (w *Worker) Add(v domain.Vault, path string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.queue.add(v, path)
}

// CountWaiting is how many sources a person named are still in line.
func (w *Worker) CountWaiting() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.queue.countWaiting()
}

// Start names one source and says whether its run began now or waits its turn
// behind the run before it.
func (w *Worker) Start(v domain.Vault, path string) windowport.StartOutcome {
	w.mu.Lock()
	w.queue.add(v, path)
	if w.isRunning {
		w.mu.Unlock()
		return windowport.Queued
	}
	w.isRunning = true
	w.runs++
	mine := w.runs
	w.mu.Unlock()

	w.Go(func() {
		defer w.Release(mine)
		w.DrainAndRelease(w.Context(), nil)
	})
	return windowport.Began
}

// Claim takes the one run this worker has, and says whether it was free.
func (w *Worker) Claim() (uint64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isRunning {
		return 0, false
	}
	w.isRunning = true
	w.runs++
	return w.runs, true
}

// Release stops the running where this run is still the one running.
func (w *Worker) Release(run uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.runs == run {
		w.isRunning = false
	}
}

// Drain runs every source in the line, in the order they were named. A source
// is out of the line before it runs. The context is asked before a source is
// taken, so what the line holds when the application closes is still in it.
func (w *Worker) Drain(ctx context.Context) {
	for {
		w.mu.Lock()
		var (
			one     wanted
			waiting bool
		)
		if ctx.Err() == nil {
			one, waiting = w.queue.take()
		}
		w.mu.Unlock()
		if !waiting {
			return
		}
		w.each(ctx, one.vault, one.path)
	}
}

// DrainAndRelease drains the line and stops the running. The line is found
// empty and the running stopped under one hold of the lock, so a source named
// at that instant is answered "began" and run by the run that answers it.
// beforeStop, when set, is called under that hold.
func (w *Worker) DrainAndRelease(ctx context.Context, beforeStop func()) {
	for {
		w.Drain(ctx)
		w.mu.Lock()
		if w.queue.countWaiting() == 0 || ctx.Err() != nil {
			if beforeStop != nil {
				beforeStop()
			}
			if w.idle != nil {
				w.idle()
			}
			w.isRunning = false
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
	}
}

// RepeatRounds runs round, waits every, and runs it again until the context
// ends.
func (w *Worker) RepeatRounds(ctx context.Context, wait machineport.Wait, every time.Duration, round func()) {
	for {
		round()
		if wait(ctx, every) != nil {
			return
		}
	}
}

// Report puts one piece of work in the list. Work a person started is shown at
// once, and work nobody asked for is shown once it has lasted.
func (w *Worker) Report(pos Task, isUserStarted bool) {
	pos.IsUserStarted = isUserStarted
	if w.tasks != nil {
		w.tasks.Set(pos)
	}
}

// Finish takes a piece of work out of the list.
func (w *Worker) Finish(id string) {
	if w.tasks != nil {
		w.tasks.Remove(id)
	}
}
