package broadcast

import (
	"errors"
	"sync"
)

// ErrClosing is returned for a write requested after the vault is sealed for
// closing. Nothing is written.
var ErrClosing = errors.New("this vault is closing")

// Inflight is the writes taken and not yet finished.
//
// What is counted is the writing itself, and not the answer travelling back to
// the client. The database closes once nothing is being written.
type Inflight struct {
	mu       sync.Mutex
	count    int
	isSealed bool
	idle     chan struct{}
	isOver   bool
}

// Begin registers a write, and rejects one that arrives after the vault is sealed.
func (w *Inflight) Begin() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isSealed {
		return false
	}
	w.count++
	return true
}

func (w *Inflight) Finish() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.count--
	w.closeIfIdle()
}

// Seal shuts the door on new writes and answers with what closes once the ones
// already taken have finished. The set it waits on is therefore finite and
// does not grow.
func (w *Inflight) Seal() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isSealed = true
	if w.idle == nil {
		w.idle = make(chan struct{})
	}
	w.closeIfIdle()
	return w.idle
}

// Open accepts writes again. The sealed vault is gone and another is now
// current.
func (w *Inflight) Open() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isSealed = false
	w.isOver = false
	w.idle = nil
}

// Count is how many writes are in flight.
func (w *Inflight) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.count
}

func (w *Inflight) closeIfIdle() {
	if w.isSealed && w.count == 0 && !w.isOver {
		w.isOver = true
		if w.idle != nil {
			close(w.idle)
		}
	}
}
