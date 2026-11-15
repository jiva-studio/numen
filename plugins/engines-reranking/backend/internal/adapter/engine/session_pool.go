package engine

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed is returned by a call on a scorer that is closed.
var ErrClosed = errors.New("the model is closed")

// sessionPool lends a session to at most lanes callers at a time.
type sessionPool[S any] struct {
	idle     chan S
	sessions []S
	lanes    int
	destroy  func(S)
	stop     chan struct{}
	once     sync.Once
}

func newSessionPool[S any](sessions []S, lanes int, destroy func(S)) *sessionPool[S] {
	lanes = max(lanes, 1)
	p := &sessionPool[S]{
		idle:     make(chan S, lanes),
		sessions: sessions,
		lanes:    lanes,
		destroy:  destroy,
		stop:     make(chan struct{}),
		once:     sync.Once{},
	}
	for lane := range lanes {
		p.idle <- sessions[lane%len(sessions)]
	}
	return p
}

func (p *sessionPool[S]) acquire(ctx context.Context) (S, error) {
	var zero S
	select {
	case <-p.stop:
		return zero, ErrClosed
	default:
	}
	select {
	case <-p.stop:
		return zero, ErrClosed
	case <-ctx.Done():
		return zero, ctx.Err()
	case s := <-p.idle:
		select {
		case <-p.stop:
			p.idle <- s
			return zero, ErrClosed
		default:
			return s, nil
		}
	}
}

func (p *sessionPool[S]) release(s S) {
	p.idle <- s
}

func (p *sessionPool[S]) close() {
	p.once.Do(func() {
		close(p.stop)
		for range p.lanes {
			<-p.idle
		}
		for _, s := range p.sessions {
			p.destroy(s)
		}
	})
}
