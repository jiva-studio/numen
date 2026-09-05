package editor

import (
	"errors"
	"sync"
)

// errClosing is what a write asked for after the door is shut gets. Nothing is
// written.
var errClosing = errors.New("this vault is closing")

// inflight is the writes taken and not yet finished.
//
// What is counted is the writing itself, and not the answer travelling back to
// the client. The database closes once nothing is being written.
type inflight struct {
	mu     sync.Mutex
	count  int
	sealed bool
	idle   chan struct{}
	over   bool
}

// begin takes a write, and refuses one that arrives after the door is shut.
func (w *inflight) begin() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.sealed {
		return false
	}
	w.count++
	return true
}

func (w *inflight) done() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.count--
	w.reckon()
}

// seal shuts the door on new writes and answers with what closes once the ones
// already taken have finished. The set it waits on is therefore finite and
// does not grow.
func (w *inflight) seal() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sealed = true
	if w.idle == nil {
		w.idle = make(chan struct{})
	}
	w.reckon()
	return w.idle
}

// open takes writes again. The vault the door was shut on is gone and another
// is in front of the person.
func (w *inflight) open() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.sealed, w.over, w.idle = false, false, nil
}

// reckon ends the wait once nothing is being written. The lock is held.
func (w *inflight) reckon() {
	if !w.sealed || w.over || w.count > 0 {
		return
	}
	w.over = true
	close(w.idle)
}
