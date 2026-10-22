package transcription

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A resident is opened transcription models kept loaded between recordings,
// together with the lock they are held under. Closing it lets the sessions go
// and then the lock.
type resident struct {
	mu       sync.Mutex
	provider port.Transcriber
	letGo    func() error
	release  func()
	isParked bool
	// generation tells a timer that fired late whether it is still the one
	// that was armed.
	generation uint64
	cancel     func() bool
}

// hold takes the models in use.
func (r *resident) hold(delta port.Transcriber, letGo func() error, release func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.provider, r.letGo, r.release = delta, letGo, release
	r.isParked = false
}

// take gives back the parked models for the next recording.
func (r *resident) take() (port.Transcriber, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.provider == nil || !r.isParked {
		return nil, false
	}
	r.isParked = false
	r.generation++
	if r.cancel != nil {
		r.cancel()
	}
	return r.provider, true
}

// park leaves the models loaded and closes them when idle passes with no take.
func (r *resident) park(delay port.Delay, idle time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isParked = true
	r.generation++
	generation := r.generation
	r.cancel = delay(idle, func() { r.expire(generation) })
}

// expire closes the models if they have stayed parked since the timer was armed.
func (r *resident) expire(generation uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.generation != generation || !r.isParked {
		return
	}
	_ = r.closeLocked()
}

// close lets the models go and then the lock. It does nothing when none are held.
func (r *resident) close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closeLocked()
}

func (r *resident) closeLocked() error {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.provider == nil {
		return nil
	}
	err := r.letGo()
	r.release()
	r.provider, r.letGo, r.release, r.isParked = nil, nil, nil, false
	r.generation++
	return err
}
