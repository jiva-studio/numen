package run

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
)

// Task is the unit of work the composition root names: it may not import internal/vaultreading.
type Task = vaultreading.Task

// Bytes is the unit of a download, which the composition root names.
const Bytes = vaultreading.Bytes

// Tasks is what the application is doing now.
//
// It is safe to use from several goroutines: work reports itself from wherever
// it runs, and what shows it runs somewhere else.
type Tasks struct {
	mu         sync.Mutex
	activeItem map[string]vaultreading.Task
	order      []string
	waiting    []chan []vaultreading.Task
}

// NewTasks is what the application is doing now.
func NewTasks() *Tasks { return &Tasks{activeItem: map[string]vaultreading.Task{}} }

// Set adds a task to the list, or replaces it if it exists.
//
// Tasks are shown in the order they were first seen, so a list that changes
// while the user is reading it does not reorder.
func (t *Tasks) Set(task vaultreading.Task) {
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

// Cancel stops the listed task with this id and reports whether it could be
// canceled. The task stays listed until its work removes it.
func (t *Tasks) Cancel(id string) bool {
	t.mu.Lock()
	task, isListed := t.activeItem[id]
	t.mu.Unlock()
	if !isListed || task.Cancel == nil {
		return false
	}
	task.Cancel()
	return true
}

// Clear takes all tasks out.
func (t *Tasks) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.activeItem) == 0 {
		return
	}
	t.activeItem = map[string]vaultreading.Task{}
	t.order = nil
	t.notifyWatchers()
}

// List is what is being done now.
func (t *Tasks) List() []vaultreading.Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.list()
}

func (t *Tasks) list() []vaultreading.Task {
	out := make([]vaultreading.Task, 0, len(t.order))
	for _, id := range t.order {
		if task, target := t.activeItem[id]; target {
			out = append(out, task)
		}
	}
	return out
}

// Watch returns the list now and every time it changes, until the context ends.
// A listener that is not keeping up is given the newest list and not a queue of
// old ones.
func (t *Tasks) Watch(ctx context.Context) <-chan []vaultreading.Task {
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 2: tell drains before it sends, so the channel never holds more than one list"
	watcher := make(chan []vaultreading.Task, 1)

	t.mu.Lock()
	t.waiting = append(t.waiting, watcher)
	watcher <- t.list()
	t.mu.Unlock()

	go func() {
		<-ctx.Done()
		t.mu.Lock()
		defer t.mu.Unlock()
		t.waiting = slices.DeleteFunc(t.waiting, func(pos chan []vaultreading.Task) bool { return pos == watcher })
		close(watcher)
	}()
	return watcher
}

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
