package onnx

import (
	"context"
	"errors"
	"sync"
)

// errClosed is what a session asked of a closed pool answers.
var errClosed = errors.New("the embedder is closed")

// sessionPool lends a session to at most lanes callers at a time. Each lane
// holds one session, and with fewer sessions than lanes the lanes share them.
// Close returns only after every lane is back, and destroys each session
// exactly once.
type sessionPool[S any] struct {
	idle     chan S
	sessions []S
	lanes    int
	destroy  func(S)
	done     chan struct{}
	once     sync.Once
}

// newSessionPool is a pool of lanes over the sessions, which are dealt to the
// lanes in turn. A pool has at least one lane.
func newSessionPool[S any](sessions []S, lanes int, destroy func(S)) *sessionPool[S] {
	lanes = max(lanes, 1)
	p := &sessionPool[S]{
		idle:     make(chan S, lanes),
		sessions: sessions,
		lanes:    lanes,
		destroy:  destroy,
		done:     make(chan struct{}),
	}
	for lane := range lanes {
		p.idle <- sessions[lane%len(sessions)]
	}
	return p
}

// acquire lends the session of a free lane, waiting for one to come back when
// all are lent. A nil ctx waits without a bound.
func (p *sessionPool[S]) acquire(ctx context.Context) (S, error) {
	var zero S
	var cancelled <-chan struct{}
	if ctx != nil {
		cancelled = ctx.Done()
	}
	select {
	case <-p.done:
		return zero, errClosed
	default:
	}
	select {
	case <-p.done:
		return zero, errClosed
	case <-cancelled:
		return zero, ctx.Err()
	case s := <-p.idle:
		select {
		case <-p.done:
			p.idle <- s
			return zero, errClosed
		default:
			return s, nil
		}
	}
}

// release takes a lent lane back.
func (p *sessionPool[S]) release(s S) {
	p.idle <- s
}

// close stops lending, waits for every lent lane to come back and destroys all
// the sessions. A later or concurrent call returns once the first has finished.
func (p *sessionPool[S]) close() {
	p.once.Do(func() {
		close(p.done)
		for range p.lanes {
			<-p.idle
		}
		for _, s := range p.sessions {
			p.destroy(s)
		}
	})
}
