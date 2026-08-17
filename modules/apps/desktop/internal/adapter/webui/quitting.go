package webui

import (
	"context"
	"strconv"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// leaving is everyone drawing this vault, for the moment the window goes.
//
// A client holds unwritten work in a buffer this process cannot see, so it is
// asked to write what it owes and answers when it has. A client that stops
// listening stops being owed.
type leaving struct {
	mu    sync.Mutex
	next  int
	pages map[string]*page
	asked bool
	idle  chan struct{}
	over  bool
}

// page is one client drawing the vault, and whether it has answered.
type page struct {
	tell     chan string
	answered bool
}

// listen registers a client, and answers with the token it will flush under,
// what it is told on, and how it stops listening.
func (l *leaving) listen() (string, <-chan string, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.pages == nil {
		l.pages = map[string]*page{}
	}
	token := strconv.Itoa(l.next)
	l.next++
	p := &page{tell: make(chan string, 1)}
	l.pages[token] = p
	if l.asked {
		p.tell <- token
	}

	return token, p.tell, func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if _, listening := l.pages[token]; !listening {
			return
		}
		delete(l.pages, token)
		close(p.tell)
		l.reckon()
	}
}

// ask tells every client to write what it owes, and answers with what closes
// once every one of them has.
func (l *leaving) ask() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.asked = true
	if l.idle == nil {
		l.idle = make(chan struct{})
	}
	for token, p := range l.pages {
		if p.answered {
			continue
		}
		select {
		case p.tell <- token:
		default:
		}
	}
	l.reckon()
	return l.idle
}

// flushed marks one client as having written what it owed.
func (l *leaving) flushed(token string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if p, listening := l.pages[token]; listening {
		p.answered = true
	}
	l.reckon()
}

// reckon ends the wait once nothing is owed. The lock is held.
func (l *leaving) reckon() {
	if !l.asked || l.over {
		return
	}
	for _, p := range l.pages {
		if !p.answered {
			return
		}
	}
	l.over = true
	close(l.idle)
}

// Quitting says the window is going, for as long as the client listens.
func (a *API) Quitting(
	ctx context.Context,
	_ *connect.Request[v1.QuittingRequest],
	out *connect.ServerStream[v1.QuittingResponse],
) error {
	token, told, done := a.Leaving.listen()
	defer done()

	// The token before anything is asked. A client that has not been given one
	// cannot answer, and a stream that says nothing until the window goes is
	// indistinguishable from one that never opened.
	if err := out.Send(&v1.QuittingResponse{Token: token}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case asked, listening := <-told:
			if !listening {
				return nil
			}
			if err := out.Send(&v1.QuittingResponse{Token: asked, Flush: true}); err != nil {
				return err
			}
		}
	}
}

// Flushed says a client has written everything it owed.
func (a *API) Flushed(
	_ context.Context,
	r *connect.Request[v1.FlushedRequest],
) (*connect.Response[v1.FlushedResponse], error) {
	a.Leaving.flushed(r.Msg.GetToken())
	return connect.NewResponse(&v1.FlushedResponse{}), nil
}
