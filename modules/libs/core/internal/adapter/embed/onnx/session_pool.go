package onnx

import (
	"context"
	"errors"
	"sync"
)

// errClosed is what a session asked of a closed pool answers.
var errClosed = errors.New("the embedder is closed")

// sessionPool lends each of its sessions to one caller at a time. Close returns
// only after every lent session is back, and destroys each exactly once.
type sessionPool[S any] struct {
	idle    chan S
	size    int
	destroy func(S)
	done    chan struct{}
	once    sync.Once
}

func newSessionPool[S any](sessions []S, destroy func(S)) *sessionPool[S] {
	p := &sessionPool[S]{
		idle:    make(chan S, len(sessions)),
		size:    len(sessions),
		destroy: destroy,
		done:    make(chan struct{}),
	}
	for _, s := range sessions {
		p.idle <- s
	}
	return p
}

// acquire lends one session, waiting for one to come back when all are lent.
// A nil ctx waits without a bound.
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

// release takes a lent session back.
func (p *sessionPool[S]) release(s S) {
	p.idle <- s
}

// close stops lending, waits for every lent session to come back and destroys
// all of them. A later or concurrent call returns once the first has finished.
func (p *sessionPool[S]) close() {
	p.once.Do(func() {
		close(p.done)
		for range p.size {
			p.destroy(<-p.idle)
		}
	})
}
