// Package task holds what the application is doing behind the window.
//
// Anything that takes long enough for a person to wonder about it puts itself
// here, and whatever shows work to a person reads the whole list. That is the
// point of a list: the next kind of work is an entry rather than another field
// on a message, another poll, and another branch in the window.
//
// A task is what is happening, not what happened. Nothing here is a record: a
// task ends by being taken out.
package task

import (
	"context"
	"slices"
	"sync"
)

// A Task is one piece of work, as a person is shown it.
type Task struct {
	// ID is what this work is called, so that the same work reported again
	// replaces itself rather than appearing twice.
	ID string

	// Doing is the work, in the words to show: "Reading a scan".
	Doing string
	// About is what it is on, and is empty for work that is on nothing in
	// particular.
	About string

	// Done and Total are how far it has got, when there is a total to count
	// against. Both zero is work that is running with nothing to count, which is
	// an ordinary state and not an unknown one.
	Done, Total int64

	// Failed is why the work stopped, when it stopped badly. A task that failed
	// stays in the list until whoever put it there takes it out, because a
	// failure nobody was shown is a failure nobody can act on.
	Failed string

	// Asked is set for work a person started and is waiting to be told about.
	// Work nobody asked for is shown once it has lasted, and most of it ends
	// before that.
	Asked bool

	// Counting is what Done and Total are counted in.
	Counting Counting
}

// Counting is what a piece of work counts. Bytes are read out in the sizes a
// person reads them in, and everything else is counted one by one.
type Counting int

const (
	Things Counting = iota
	Bytes
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

func New() *Tasks { return &Tasks{held: map[string]Task{}} }

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
	if _, held := t.held[task.ID]; !held {
		t.order = append(t.order, task.ID)
	}
	t.held[task.ID] = task
	t.tell()
}

// Done takes one task out. A task that is not there is the outcome asked for.
func (t *Tasks) Done(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, held := t.held[id]; !held {
		return
	}
	delete(t.held, id)
	t.order = slices.DeleteFunc(t.order, func(at string) bool { return at == id })
	t.tell()
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
		if task, held := t.held[id]; held {
			out = append(out, task)
		}
	}
	return out
}

// Watch is the list, now and every time it changes, until the context ends.
//
// It is a stream rather than a question asked over and over: what is being done
// is known here the moment it changes, and a window that asks on a timer is a
// window that is either late or asking for nothing.
//
// A listener that is not keeping up is given the newest list and not a queue of
// old ones: what the work was doing a second ago is of no interest to anybody.
func (t *Tasks) Watch(ctx context.Context) <-chan []Task {
	told := make(chan []Task, 1)

	t.mu.Lock()
	t.waiting = append(t.waiting, told)
	told <- t.list()
	t.mu.Unlock()

	go func() {
		<-ctx.Done()
		t.mu.Lock()
		defer t.mu.Unlock()
		t.waiting = slices.DeleteFunc(t.waiting, func(at chan []Task) bool { return at == told })
		close(told)
	}()
	return told
}

// tell hands the list to everybody watching, dropping what a listener has not
// read yet.
//
// The caller holds the lock from the change through the telling, so the last
// list a listener is left holding is the newest one.
func (t *Tasks) tell() {
	list := t.list()
	for _, told := range t.waiting {
		select {
		case <-told:
		default:
		}
		select {
		case told <- list:
		default:
		}
	}
}
