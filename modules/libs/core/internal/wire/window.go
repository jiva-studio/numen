package wire

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// The windows this application opens. A question is asked of one of them by
// name, so that the two, which are open on the same vault at once, are never
// answered for each other.
const (
	// Editor is the window a person writes in.
	Editor = "editor"
	// Review is the window a person runs their cards in.
	Review = "review"
)

// ErrAnotherWindow is a question naming a window other than the one it reached.
var ErrAnotherWindow = errors.New("that is another window")

// Window is one window of this application, as the schema answers about it:
// everything being done behind it, and everyone drawing it for the moment it
// goes.
//
// Both windows answer this from here, so what is being done reaches a person in
// one shape and the drain that holds a window back is one piece of code.
type Window struct {
	// Named is the window this answers for, which is Editor or Review.
	Named string

	// Tasking is everything being done behind the window. A window holding none
	// says there is nothing.
	Tasking *task.Tasks

	// Vault is the identity of the vault this window has in front of the
	// person. Nil is a window open on the installation rather than on any one
	// vault, which answers with none.
	Vault func() string

	clients leaving
}

// Again is how often a stream says what it last said when nothing has changed.
//
// A window is handed each list by the write that follows it. Nothing else tells
// a handler its client has gone: the request context belongs to the process and
// is cancelled when the window closes, not when a page is reloaded away from
// under a stream. A write that fails is the one report there is, so every
// stream makes one whether or not it has anything to say.
const Again = time.Second

// Tasks reports everything being done behind the window, for as long as the
// client listens.
//
// The whole list goes every time any of it changes, and the first goes at once:
// a window that opened while work was running has to be told about it, and a
// stream that says nothing until something changes is indistinguishable from one
// that never opened.
func (w *Window) Tasks(
	ctx context.Context,
	r *connect.Request[v1.TasksRequest],
	out *connect.ServerStream[v1.TasksResponse],
) error {
	if err := w.answers(r.Msg.GetWindow()); err != nil {
		return err
	}
	if w.Tasking == nil {
		// A window that does nothing behind itself still answers, so that it has
		// one thing to listen to rather than two ways of finding out whether it
		// should.
		return out.Send(&v1.TasksResponse{})
	}

	watching, stop := context.WithCancel(ctx)
	defer stop()

	watch := w.Tasking.Watch(watching)
	repeat := time.NewTicker(Again)
	defer repeat.Stop()

	var last []task.Task
	for {
		select {
		case list, standing := <-watch:
			if !standing {
				return nil
			}
			if err := out.Send(&v1.TasksResponse{Tasks: doing(list)}); err != nil {
				return err
			}
			last = list
			repeat.Reset(Again)

		case <-repeat.C:
			if err := out.Send(&v1.TasksResponse{Tasks: doing(last)}); err != nil {
				return err
			}
		}
	}
}

// Quitting says the window is going, for as long as the client listens.
func (w *Window) Quitting(
	ctx context.Context,
	r *connect.Request[v1.QuittingRequest],
	out *connect.ServerStream[v1.QuittingResponse],
) error {
	if err := w.answers(r.Msg.GetWindow()); err != nil {
		return err
	}
	token, told, done := w.clients.listen()
	defer done()

	// The token before anything is asked. A client that has not been given one
	// cannot answer, and a stream that says nothing until the window goes is
	// indistinguishable from one that never opened.
	if err := out.Send(&v1.QuittingResponse{Token: token}); err != nil {
		return err
	}

	repeat := time.NewTicker(Again)
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
func (w *Window) Flushed(
	_ context.Context,
	r *connect.Request[v1.FlushedRequest],
) (*connect.Response[v1.FlushedResponse], error) {
	if err := w.answers(r.Msg.GetWindow()); err != nil {
		return nil, err
	}
	w.clients.flushed(r.Msg.GetToken(), left(r.Msg.GetOwed()))
	return connect.NewResponse(&v1.FlushedResponse{}), nil
}

// Showing is the vault this window has in front of the person.
func (w *Window) Showing(
	_ context.Context,
	r *connect.Request[v1.ShowingRequest],
) (*connect.Response[v1.ShowingResponse], error) {
	if err := w.answers(r.Msg.GetWindow()); err != nil {
		return nil, err
	}
	out := &v1.ShowingResponse{}
	if w.Vault != nil {
		out.Vault = w.Vault()
	}
	return connect.NewResponse(out), nil
}

// answers says whether a question reached the window it names.
func (w *Window) answers(named string) error {
	if named == w.Named {
		return nil
	}
	return connect.NewError(connect.CodeNotFound, ErrAnotherWindow)
}

// Say puts one piece of work in the list.
func (w *Window) Say(at task.Task) {
	if w != nil && w.Tasking != nil {
		w.Tasking.Set(at)
	}
}

// Finished takes one piece of work out of the list.
func (w *Window) Finished(id string) {
	if w != nil && w.Tasking != nil {
		w.Tasking.Done(id)
	}
}

// Settling asks every page to write what it owes and waits for the round to
// end. It answers false where a page is holding work a person is being asked
// about, and where another round has taken this one's place.
func (w *Window) Settling(ctx context.Context) bool {
	round := w.clients.ask()
	select {
	case <-round.written:
	case <-round.questions:
	case <-round.over:
	case <-ctx.Done():
	}
	return !round.standing() && w.clients.current() == round
}

// Answered waits for the round in progress to end with every page having
// written what it owes.
func (w *Window) Answered(ctx context.Context) bool {
	round := w.clients.current()
	if round == nil {
		return false
	}
	select {
	case <-round.written:
		return true
	case <-round.over:
		return false
	case <-ctx.Done():
		return false
	}
}

// Over ends the round that was running, and lets go of whoever was waiting on
// it. What the pages hold from here belongs to the vault in front of them.
func (w *Window) Over() { w.clients.over() }

// doing is the work as the schema says it.
func doing(list []task.Task) []*v1.Task {
	out := make([]*v1.Task, 0, len(list))
	for _, at := range list {
		out = append(out, &v1.Task{
			Id:       at.ID,
			Doing:    at.Doing,
			About:    at.About,
			Done:     at.Count,
			Total:    at.Total,
			Failed:   at.Failed,
			Asked:    at.Asked,
			Counting: counted(at.Unit),
		})
	}
	return out
}

// counted is what a task counts, as the schema says it.
func counted(in task.Unit) v1.Counting {
	switch in {
	case task.Bytes:
		return v1.Counting_COUNTING_BYTES
	case task.Seconds:
		return v1.Counting_COUNTING_SECONDS
	case task.Things:
		return v1.Counting_COUNTING_THINGS
	}
	return v1.Counting_COUNTING_THINGS
}

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

// left is what a client said it has left, in the words this uses. A client with
// nothing left and one that has written everything both leave the window free
// to go; one that named nothing has said nothing, and the round waits its bound
// for it the way it waits for a client that never answered.
func left(said v1.Owed) owed {
	switch said {
	case v1.Owed_OWED_NOTHING, v1.Owed_OWED_WRITTEN:
		return wrote
	case v1.Owed_OWED_ASKING:
		return asks
	}
	return silent
}

// leaving is everyone drawing this window, for the moment it goes.
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

// client is one browser tab drawing the window, and what it has said.
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
// it.
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
