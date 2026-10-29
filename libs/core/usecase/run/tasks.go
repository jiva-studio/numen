package run

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/internal/vaultreading"
)

// A Task is one unit of work, as displayed to the user.
type Task = vaultreading.Task

// Unit is what a piece of work counts. Bytes are read out in the sizes a
// person reads them in, and everything else is counted one by one.
type Unit = vaultreading.Unit

const (
	Things  = vaultreading.Things
	Bytes   = vaultreading.Bytes
	Seconds = vaultreading.Seconds
)

// Tasks is what the application is doing now.
//
// It is safe to use from several goroutines: work reports itself from wherever
// it runs, and what shows it runs somewhere else.
type Tasks struct {
	mu         sync.Mutex
	activeItem map[string]Task
	order      []string
	waiting    []chan []Task
}

// NewTasks is what the application is doing now.
func NewTasks() *Tasks { return &Tasks{activeItem: map[string]Task{}} }

// Set adds a task to the list, or replaces it if it exists.
//
// Tasks are shown in the order they were first seen, so a list that changes
// while the user is reading it does not reorder.
func (t *Tasks) Set(task Task) {
	if task.ID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, target := t.activeItem[task.ID]; !target {
		t.order = append(t.order, task.ID)
	}
	t.activeItem[task.ID] = task
	t.notifyWatchers()
}

// Remove deletes a task. Removing a task that does not exist is not an error.
func (t *Tasks) Remove(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, target := t.activeItem[id]; !target {
		return
	}
	delete(t.activeItem, id)
	t.order = slices.DeleteFunc(t.order, func(pos string) bool { return pos == id })
	t.notifyWatchers()
}

// Clear takes all tasks out.
func (t *Tasks) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.activeItem) == 0 {
		return
	}
	t.activeItem = map[string]Task{}
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
		if task, target := t.activeItem[id]; target {
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
