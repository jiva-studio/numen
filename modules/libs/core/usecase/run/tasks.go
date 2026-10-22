package run

import (
	"context"
	"slices"
	"sync"
)

// A Task is one piece of work, as a person is shown it.
type Task struct {
	// ID is what this work is called, so that the same work reported again
	// replaces itself and not appearing twice.
	ID string

	// Title is the work, in the words to show: "Recognising a scan".
	Title string
	// About is what it is on, and is empty for work that is on nothing in
	// particular.
	About string

	// Count and Total are how far it has got, when there is a total to count
	// against. Both zero is work that is running with nothing to count, which is
	// an ordinary state and not an unknown one.
	Count, Total int64

	// Error is why the work stopped, when it stopped badly. A task that stopped
	// badly stays in the list until whoever put it there takes it out.
	Error string

	// IsUserStarted is set for work a person started and is waiting to be told about.
	// Work nobody asked for is shown once it has lasted, and most of it ends
	// before that.
	IsUserStarted bool

	// Unit is what Count and Total are counted in.
	Unit Unit
}

// Unit is what a piece of work counts. Bytes are read out in the sizes a
// person reads them in, and everything else is counted one by one.
type Unit int

const (
	Things Unit = iota
	Bytes
	// Seconds are a stretch of a recording, read out as a length of time.
	Seconds
)

// Tasks is what the application is doing now.
//
// It is safe to use from several goroutines: work reports itself from wherever
// it runs, and what shows it runs somewhere else.
type Tasks struct {
	mu      sync.Mutex
	held    map[string]Task
	order   []string
	waiting []chan []Task
}

// NewTasks is what the application is doing now.
func NewTasks() *Tasks { return &Tasks{held: map[string]Task{}} }

// New is an alias for NewTasks.
func New() *Tasks { return NewTasks() }

// Set puts one task in the list, or replaces it where it is.
//
// The order tasks were first seen in is the order they are shown in, so a list
// that changes while a person is reading it does not rearrange itself under
// them.
func (t *Tasks) Set(task Task) {
	if task.ID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, target := t.held[task.ID]; !target {
		t.order = append(t.order, task.ID)
	}
	t.held[task.ID] = task
	t.notifyWatchers()
}

// Remove takes one task out. A task that is not there is the outcome asked for.
func (t *Tasks) Remove(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, target := t.held[id]; !target {
		return
	}
	delete(t.held, id)
	t.order = slices.DeleteFunc(t.order, func(pos string) bool { return pos == id })
	t.notifyWatchers()
}

// Clear takes all tasks out.
func (t *Tasks) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.held) == 0 {
		return
	}
	t.held = map[string]Task{}
	t.order = nil
	t.notifyWatchers()
}

// List is what is being done now.
func (t *Tasks) List() []Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.list()
}

// list is the tasks in the order they were first seen. The caller holds the
// lock.
func (t *Tasks) list() []Task {
	out := make([]Task, 0, len(t.order))
	for _, id := range t.order {
		if task, target := t.held[id]; target {
			out = append(out, task)
		}
	}
	return out
}

// Watch is the list, now and every time it changes, until the context ends.
//
// What is being done is known here the moment it changes.
//
// A listener that is not keeping up is given the newest list and not a queue of
// old ones: what the work was doing a second ago is of no interest to anybody.
func (t *Tasks) Watch(ctx context.Context) <-chan []Task {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 2: tell drains before it sends, so the channel never holds more than one list"
	watcher := make(chan []Task, 1)

	t.mu.Lock()
	t.waiting = append(t.waiting, watcher)
	watcher <- t.list()
	t.mu.Unlock()

	go func() {
		<-ctx.Done()
		t.mu.Lock()
		defer t.mu.Unlock()
		t.waiting = slices.DeleteFunc(t.waiting, func(pos chan []Task) bool { return pos == watcher })
		close(watcher)
	}()
	return watcher
}

// notifyWatchers hands the list to everybody watching, dropping what a listener has not
// read yet.
//
// The caller holds the lock from the change through the telling, so the last
// list a listener is left holding is the newest one.
func (t *Tasks) notifyWatchers() {
	list := t.list()
	for _, watcher := range t.waiting {
		select {
		case <-watcher:
		default:
		}
		select {
		case watcher <- list:
		default:
		}
	}
}
