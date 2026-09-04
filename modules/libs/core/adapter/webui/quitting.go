package webui

import (
	"context"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// owed is what a page has left when it answers.
type owed int

const (
	// silent is a page that has said nothing about this round.
	silent owed = iota
	// wrote is a page with nothing left: what it held is on disk.
	wrote
	// asks is a page holding work that could not be written, with a person
	// being asked what happens to it.
	asks
)

// leaving is everyone drawing this vault, for the moment the window goes.
//
// A client holds unwritten work in a buffer this process cannot see, so it is
// asked to write what it owes and answers when it has. A client that stops
// listening stops being owed, unless its stream ended with a question standing:
// the work is still in that buffer, and it is silence until a page listens
// again.
type leaving struct {
	mu    sync.Mutex
	next  int
	pages map[string]*client
	round *round
}

// client is one browser tab drawing the vault, and what it has said.
type client struct {
	told chan string
	said owed
	// gone is a page whose stream ended with a question standing. It is told
	// nothing and answers nothing; a page that listens takes it over.
	gone bool
}

// round is one asking of every page, and how far it has got.
type round struct {
	// written closes once every page has written what it owes.
	written chan struct{}
	// questions closes once a page is holding work a person has to answer for.
	questions chan struct{}
	// over closes once another round has taken this one's place.
	over chan struct{}

	// The lock of the leaving that made this round is held for these.
	settled bool
	raised  bool
	past    bool
}

// standing reports whether a question a person has to answer is outstanding.
func (r *round) standing() bool {
	select {
	case <-r.questions:
		return true
	default:
		return false
	}
}

// listen registers a client, and answers with the token it will flush under,
// what it is told on, and how it stops listening.
func (l *leaving) listen() (string, <-chan string, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.pages == nil {
		l.pages = map[string]*client{}
	}
	// One window draws one vault, so a page that listens is the page that went,
	// and it takes over what that one was holding.
	for token, p := range l.pages {
		if p.gone {
			delete(l.pages, token)
		}
	}
	token := strconv.Itoa(l.next)
	l.next++
	p := &client{told: make(chan string, 1)}
	l.pages[token] = p
	if l.asking() {
		p.told <- token
	}
	l.reckon()

	return token, p.told, func() { l.left(token, p) }
}

// asking reports whether a round is running. The lock is held.
func (l *leaving) asking() bool {
	return l.round != nil && !l.round.past
}

// left is one client no longer listening.
func (l *leaving) left(token string, p *client) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.pages[token] != p || p.gone {
		return
	}
	close(p.told)
	// A page that went with a question standing is the only place that work
	// exists. It is remembered, and it is remembered as silence, which is what
	// a round waits its bound for.
	if p.said == asks {
		p.said = silent
		p.gone = true
	} else {
		delete(l.pages, token)
	}
	l.reckon()
}

// ask begins a round: every page is told to write what it owes, and what it
// said in the round before counts for nothing. A round already running is put
// past, and whoever was waiting on it is let go.
func (l *leaving) ask() *round {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.asking() {
		l.round.past = true
		close(l.round.over)
	}
	l.round = &round{
		written:   make(chan struct{}),
		questions: make(chan struct{}),
		over:      make(chan struct{}),
	}
	for token, p := range l.pages {
		p.said = silent
		if p.gone {
			continue
		}
		select {
		case p.told <- token:
		default:
		}
	}
	l.reckon()
	return l.round
}

// over ends the round that was running, and lets go of whoever was waiting on
// it. What the pages hold from here belongs to the vault in front of them.
func (l *leaving) over() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.asking() {
		return
	}
	l.round.past = true
	close(l.round.over)
	l.round = nil
}

// current is the round running, or nothing.
func (l *leaving) current() *round {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.asking() {
		return nil
	}
	return l.round
}

// flushed records what one page has left.
func (l *leaving) flushed(token string, said owed) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if p, listening := l.pages[token]; listening && !p.gone {
		p.said = said
	}
	l.reckon()
}

// reckon says how far the round has got: everything is written, or a question
// a person has to answer is outstanding. The lock is held.
func (l *leaving) reckon() {
	r := l.round
	if r == nil || r.past {
		return
	}
	for _, p := range l.pages {
		if p.said == asks && !r.raised {
			r.raised = true
			close(r.questions)
		}
	}
	if r.settled {
		return
	}
	for _, p := range l.pages {
		if p.said != wrote {
			return
		}
	}
	r.settled = true
	close(r.written)
}

// Quitting says the window is going, for as long as the client listens.
func (a *API) Quitting(
	ctx context.Context,
	_ *connect.Request[v1.QuittingRequest],
	out *connect.ServerStream[v1.QuittingResponse],
) error {
	token, told, done := a.clients.listen()
	defer done()

	// The token before anything is asked. A client that has not been given one
	// cannot answer, and a stream that says nothing until the window goes is
	// indistinguishable from one that never opened.
	if err := out.Send(&v1.QuittingResponse{Token: token}); err != nil {
		return err
	}

	repeat := time.NewTicker(again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// The same token, and nothing asked of the client. A page that has
			// gone fails the write, and this stream is what holds the window
			// back at the quit.
			if err := out.Send(&v1.QuittingResponse{Token: token}); err != nil {
				return err
			}
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

// Flushed says what a client has left.
func (a *API) Flushed(
	_ context.Context,
	r *connect.Request[v1.FlushedRequest],
) (*connect.Response[v1.FlushedResponse], error) {
	a.clients.flushed(r.Msg.GetToken(), left(r.Msg.GetOwed()))
	return connect.NewResponse(&v1.FlushedResponse{}), nil
}

// left is what a client said it has left, in the words this uses. A client with
// nothing left and one that has written everything both leave the window free
// to go.
func left(said v1.Owed) owed {
	if said == v1.Owed_OWED_ASKING {
		return asks
	}
	return wrote
}
