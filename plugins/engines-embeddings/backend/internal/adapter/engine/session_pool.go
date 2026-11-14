package engine

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

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
	var canceled <-chan struct{}
	if ctx != nil {
		canceled = ctx.Done()
	}
	select {
	case <-p.stop:
		// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
		return zero, ports.ErrClosed
	default:
	}
	select {
	case <-p.stop:
		// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
		return zero, ports.ErrClosed
	case <-canceled:
		// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
		return zero, ctx.Err()
	case s := <-p.idle:
		select {
		case <-p.stop:
			p.idle <- s
			// gomutants:disable-next-line RETURN_ZERO reason="zero is the zero value of S, identical to *new(S)"
			return zero, ports.ErrClosed
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
