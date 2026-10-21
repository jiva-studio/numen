package engine

import (
	"context"
	"errors"
	"sync"
)

var errClosed = errors.New("the embedder is closed")

// sessionPool lends a session to at most lanes callers at a time.
type sessionPool[S any] struct {
	idle     chan S
	sessions []S
	lanes    int
	destroy  func(S)
	done     chan struct{}
	once     sync.Once
}

// newSessionPool creates a pool of lanes over the sessions.
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

// acquire lends the session of a free lane.
func (p *sessionPool[S]) acquire(ctx context.Context) (S, error) {
	var zero S
	var cancelled <-chan struct{}
	if ctx != nil {
		cancelled = ctx.Done()
	}
	select {
	case <-p.done:
		// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
		return zero, errClosed
	default:
	}
	select {
	case <-p.done:
		// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
		return zero, errClosed
	case <-cancelled:
		// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
		return zero, ctx.Err()
	case s := <-p.idle:
		select {
		case <-p.done:
			p.idle <- s
			// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
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

// close stops lending and destroys all sessions.
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
