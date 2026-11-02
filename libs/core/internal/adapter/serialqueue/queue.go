package serialqueue

import "sync"

// Queue runs the functions given to it one after another. A change that takes seconds, such as a
// plugin starting or stopping, runs there after the call that asked for it has answered.
type Queue struct {
	mu   sync.Mutex
	last chan struct{}
}

// New returns an empty queue.
func New() *Queue {
	return &Queue{mu: sync.Mutex{}, last: nil}
}

// Go runs fn after every function given before it has returned.
func (q *Queue) Go(fn func()) {
	signal := make(chan struct{})
	q.mu.Lock()
	previous := q.last
	q.last = signal
	q.mu.Unlock()
	go func() {
		defer close(signal)
		if previous != nil {
			<-previous
		}
		fn()
	}()
}
