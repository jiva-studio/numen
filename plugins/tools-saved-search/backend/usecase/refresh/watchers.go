package refresh

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// changeBuffer is how many changes a watch holds that its reader has not taken; a change beyond them is
// dropped, and the reader reads the state after the ones it holds.
const changeBuffer = 16

type watchKey struct {
	vault domain.VaultID
	path  string
}

// watchers are the watches of the notes, the notes whose run is on its way and the notes without a good
// run that a batch is to run.
type watchers struct {
	mu        sync.Mutex
	next      int
	listeners map[watchKey]map[int]chan Change
	running   map[watchKey]int
	queued    map[watchKey]bool
}

func newWatchers() *watchers {
	return &watchers{
		mu: sync.Mutex{}, next: 0, listeners: map[watchKey]map[int]chan Change{}, running: map[watchKey]int{},
		queued: map[watchKey]bool{},
	}
}

func (w *watchers) add(key watchKey, onLast func()) (<-chan Change, func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	id := w.next
	w.next++
	changes := make(chan Change, changeBuffer)
	if w.listeners[key] == nil {
		w.listeners[key] = map[int]chan Change{}
	}
	w.listeners[key][id] = changes
	var once sync.Once
	return changes, func() {
		once.Do(func() {
			w.mu.Lock()
			defer w.mu.Unlock()
			delete(w.listeners[key], id)
			if len(w.listeners[key]) == 0 {
				delete(w.listeners, key)
				onLast()
			}
			close(changes)
		})
	}
}

func (w *watchers) begin(key watchKey) {
	w.mu.Lock()
	defer w.mu.Unlock()
	wasRunning := w.isRunningLocked(key)
	w.running[key]++
	if !wasRunning {
		w.sendLocked(key, Change{ComputedAt: time.Time{}, IsRunning: true})
	}
}

func (w *watchers) end(key watchKey, computedAt time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running[key]--
	if w.running[key] <= 0 {
		delete(w.running, key)
	}
	if !computedAt.IsZero() {
		delete(w.queued, key)
	}
	if isRunning := w.isRunningLocked(key); !isRunning || !computedAt.IsZero() {
		w.sendLocked(key, Change{ComputedAt: computedAt, IsRunning: isRunning})
	}
}

func (w *watchers) queue(key watchKey) {
	w.mu.Lock()
	defer w.mu.Unlock()
	wasRunning := w.isRunningLocked(key)
	w.queued[key] = true
	if !wasRunning {
		w.sendLocked(key, Change{ComputedAt: time.Time{}, IsRunning: true})
	}
}

func (w *watchers) dequeue(key watchKey) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.queued[key] {
		return
	}
	delete(w.queued, key)
	if !w.isRunningLocked(key) {
		w.sendLocked(key, Change{ComputedAt: time.Time{}, IsRunning: false})
	}
}

func (w *watchers) isRunning(key watchKey) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isRunningLocked(key)
}

func (w *watchers) isRunningLocked(key watchKey) bool {
	return w.running[key] > 0 || w.queued[key]
}

func (w *watchers) isWatched(key watchKey) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.listeners[key]) > 0
}

func (w *watchers) sendLocked(key watchKey, change Change) {
	for _, changes := range w.listeners[key] {
		select {
		case changes <- change:
		default:
		}
	}
}
