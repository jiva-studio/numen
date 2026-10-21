package wire

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
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

// A Reason is why something a window shows is not so, currently now, said by
// whatever found out and read by whoever draws the window.
//
// The empty reason is nothing being the matter. It is a reason and not a flag:
// a window is told what went wrong, not that something did.
type Reason struct{ why atomic.Value }

// Store says why, or the empty string once nothing is the matter.
func (r *Reason) Store(why string) { r.why.Store(why) }

// Why is what was last said, and empty while nothing is the matter.
func (r *Reason) Why() string {
	why, _ := r.why.Load().(string)
	return why
}

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
	Tasking *run.Tasks

	// Vault is the identity of the vault this window has in front of the
	// person. Nil is a window open on the installation, which answers with
	// none.
	Vault func() string

	clients flushCoordinator
}

// Again is how often a stream says what it last said when nothing has changed.
//
// A window is handed each list by the write that follows it. Nothing else tells
// a handler its client has gone: the request context belongs to the process and
// is cancelled when the window closes, not when a page is reloaded away from
// under a stream. A write that fails is the one report there is, so every
// stream makes one whether or not it has anything to say.
const Again = time.Second

// WatchTasks reports everything being done behind the window, for as long as
// the client listens.
//
// The whole list goes every time any of it changes, and the first goes at once:
// a window that opened while work was running has to be told about it, and a
// stream that says nothing until something changes is indistinguishable from one
// that never opened.
func (w *Window) WatchTasks(
	ctx context.Context,
	r *connect.Request[v1.WatchTasksRequest],
	out *connect.ServerStream[v1.WatchTasksResponse],
) error {
	if err := w.checkWindowName(r.Msg.GetWindow()); err != nil {
		return err
	}
	if w.Tasking == nil {
		// A window that does nothing behind itself still answers, so a client
		// has one thing to listen to.
		return out.Send(&v1.WatchTasksResponse{})
	}

	watching, stop := context.WithCancel(ctx)
	defer stop()

	watch := w.Tasking.Watch(watching)
	repeat := time.NewTicker(Again)
	defer repeat.Stop()

	var last []run.Task
	for {
		select {
		case list, ok := <-watch:
			if !ok {
				return nil
			}
			if err := out.Send(&v1.WatchTasksResponse{Tasks: describeTasks(list)}); err != nil {
				return err
			}
			last = list
			repeat.Reset(Again)

		case <-repeat.C:
			if err := out.Send(&v1.WatchTasksResponse{Tasks: describeTasks(last)}); err != nil {
				return err
			}
		}
	}
}

// WatchQuit says the window is going, for as long as the client listens.
func (w *Window) WatchQuit(
	ctx context.Context,
	r *connect.Request[v1.WatchQuitRequest],
	out *connect.ServerStream[v1.WatchQuitResponse],
) error {
	if err := w.checkWindowName(r.Msg.GetWindow()); err != nil {
		return err
	}
	token, told, done := w.clients.registerClient()
	defer done()

	// The token before anything is asked. A client that has not been given one
	// cannot answer, and a stream that says nothing until the window goes is
	// indistinguishable from one that never opened.
	if err := out.Send(&v1.WatchQuitResponse{Token: token}); err != nil {
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
			if err := out.Send(&v1.WatchQuitResponse{Token: token}); err != nil {
				return err
			}
		case asked, listening := <-told:
			if !listening {
				return nil
			}
			if err := out.Send(&v1.WatchQuitResponse{Token: asked, Flush: true}); err != nil {
				return err
			}
		}
	}
}

// ReportFlush says what a client has left.
func (w *Window) ReportFlush(
	_ context.Context,
	r *connect.Request[v1.ReportFlushRequest],
) (*connect.Response[v1.ReportFlushResponse], error) {
	if err := w.checkWindowName(r.Msg.GetWindow()); err != nil {
		return nil, err
	}
	w.clients.recordFlush(r.Msg.GetToken(), parseFlushResult(r.Msg.GetResult()))
	return connect.NewResponse(&v1.ReportFlushResponse{}), nil
}

// GetCurrentVault is the vault this window has in front of the person.
func (w *Window) GetCurrentVault(
	_ context.Context,
	r *connect.Request[v1.GetCurrentVaultRequest],
) (*connect.Response[v1.GetCurrentVaultResponse], error) {
	if err := w.checkWindowName(r.Msg.GetWindow()); err != nil {
		return nil, err
	}
	out := &v1.GetCurrentVaultResponse{}
	if w.Vault != nil {
		out.Vault = w.Vault()
	}
	return connect.NewResponse(out), nil
}

// checkWindowName says whether a question reached the window it names.
func (w *Window) checkWindowName(named string) error {
	if named == w.Named {
		return nil
	}
	return connect.NewError(connect.CodeNotFound, ErrAnotherWindow)
}

// AddTask puts one piece of work in the list.
func (w *Window) AddTask(at run.Task) {
	if w != nil && w.Tasking != nil {
		w.Tasking.Set(at)
	}
}

// RemoveTask takes one piece of work out of the list.
func (w *Window) RemoveTask(id string) {
	if w != nil && w.Tasking != nil {
		w.Tasking.Remove(id)
	}
}

// RequestFlush asks every page to write what it owes and waits for the request to
// end. It answers false where a page is holding work a person is being asked
// about, and where another request has taken this one's place.
func (w *Window) RequestFlush(ctx context.Context) bool {
	request := w.clients.startRequest()
	select {
	case <-request.allWritten:
	case <-request.questionRaised:
	case <-request.superseded:
	case <-ctx.Done():
	}
	return !request.isPending() && w.clients.getRunning() == request
}

// WaitForFlush waits for the request in progress to end with every page having
// written what it owes.
func (w *Window) WaitForFlush(ctx context.Context) bool {
	request := w.clients.getRunning()
	if request == nil {
		return false
	}
	select {
	case <-request.allWritten:
		return true
	case <-request.superseded:
		return false
	case <-ctx.Done():
		return false
	}
}

// CancelFlush ends the request that was running, and lets go of whoever was waiting
// on it. What the pages hold from here belongs to the vault in front of them.
func (w *Window) CancelFlush() { w.clients.endRequest() }

// describeTasks is the work as the schema says it.
func describeTasks(list []run.Task) []*v1.Task {
	out := make([]*v1.Task, 0, len(list))
	for _, at := range list {
		out = append(out, &v1.Task{
			Id:            at.ID,
			Title:         at.Title,
			About:         at.About,
			Done:          at.Count,
			Total:         at.Total,
			Error:         at.Error,
			IsUserStarted: at.IsUserStarted,
			Unit:          convertUnit(at.Unit),
		})
	}
	return out
}

// convertUnit is what a task counts, as the schema says it.
func convertUnit(in run.Unit) v1.Unit {
	switch in {
	case run.Bytes:
		return v1.Unit_UNIT_BYTES
	case run.Seconds:
		return v1.Unit_UNIT_SECONDS
	case run.Things:
		return v1.Unit_UNIT_THINGS
	}
	return v1.Unit_UNIT_THINGS
}

// flushState is what a page has left when it answers.
type flushState int

const (
	// flushUnanswered is a page that has said nothing about this request.
	flushUnanswered flushState = iota
	// flushWritten is a page with nothing left: what it held is on disk.
	flushWritten
	// flushAsking is a page holding work that could not be written, with a person
	// being asked what happens to it.
	flushAsking
)

// parseFlushResult converts a client's flush result to the internal owed state.
func parseFlushResult(result v1.FlushResult) flushState {
	switch result {
	case v1.FlushResult_FLUSH_RESULT_NOTHING, v1.FlushResult_FLUSH_RESULT_WRITTEN:
		return flushWritten
	case v1.FlushResult_FLUSH_RESULT_ASKING:
		return flushAsking
	default:
		return flushUnanswered
	}
}

// flushCoordinator is everyone drawing this window, for the moment it goes.
//
// A client holds unwritten work in a buffer this process cannot see, so it is
// asked to write what it owes and answers when it has. A client that stops
// listening stops being owed, unless its stream ended with a question standing:
// the work is still in that buffer, and it is silence until a page listens
// again.
type flushCoordinator struct {
	mu      sync.Mutex
	next    int
	clients map[string]*client
	current *flushRequest
}

// client is one browser tab drawing the window, and what it has said.
type client struct {
	flushTokens chan string
	state       flushState
	// isDisconnected is a page whose stream ended with a question standing. It is told
	// nothing and answers nothing; a page that listens takes it over.
	isDisconnected bool
}

// flushRequest is one asking of every page, and how far it has got.
type flushRequest struct {
	// allWritten closes once every page has allWritten what it owes.
	allWritten chan struct{}
	// questionRaised closes once a page is holding work a person has to answer for.
	questionRaised chan struct{}
	// superseded closes once another request has taken this one's place.
	superseded chan struct{}

	// The lock of the flush coordinator that made this request is held for these.
	isAllWrittenSignaled bool
	isQuestionSignaled   bool
	isSuperseded         bool
}

// isPending reports whether a question a person has to answer is outstanding.
func (r *flushRequest) isPending() bool {
	select {
	case <-r.questionRaised:
		return true
	default:
		return false
	}
}

// registerClient registers a client, and answers with the token it will flush under,
// what it is told on, and how it stops listening.
func (l *flushCoordinator) registerClient() (string, <-chan string, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clients == nil {
		l.clients = map[string]*client{}
	}
	// One window draws one vault, so a page that listens is the page that went,
	// and it takes over what that one was holding.
	for token, p := range l.clients {
		if p.isDisconnected {
			delete(l.clients, token)
		}
	}
	token := strconv.Itoa(l.next)
	l.next++
	p := &client{flushTokens: make(chan string, 1)}
	l.clients[token] = p
	if l.isRunning() {
		p.flushTokens <- token
	}
	l.updateProgress()

	return token, p.flushTokens, func() { l.disconnectClient(token, p) }
}

// isRunning reports whether a request is running. The lock is held.
func (l *flushCoordinator) isRunning() bool {
	return l.current != nil && !l.current.isSuperseded
}

// disconnectClient removes or marks gone one client no longer listening.
func (l *flushCoordinator) disconnectClient(token string, p *client) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clients[token] != p || p.isDisconnected {
		return
	}
	close(p.flushTokens)
	// A page that went with a question standing is the only place that work
	// exists. It is remembered, and it is remembered as silence, which is what
	// a request waits its bound for.
	if p.state == flushAsking {
		p.state = flushUnanswered
		p.isDisconnected = true
	} else {
		delete(l.clients, token)
	}
	l.updateProgress()
}

// startRequest begins a request: every page is told to write what it owes, and what it
// said in the request before counts for nothing. A request already running is put
// past, and whoever was waiting on it is let go.
func (l *flushCoordinator) startRequest() *flushRequest {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isRunning() {
		l.current.isSuperseded = true
		close(l.current.superseded)
	}
	l.current = &flushRequest{
		allWritten:     make(chan struct{}),
		questionRaised: make(chan struct{}),
		superseded:     make(chan struct{}),
	}
	for token, p := range l.clients {
		p.state = flushUnanswered
		if p.isDisconnected {
			continue
		}
		select {
		case p.flushTokens <- token:
		default:
		}
	}
	l.updateProgress()
	return l.current
}

// endRequest ends the request that was running, and lets go of whoever was waiting
// on it.
func (l *flushCoordinator) endRequest() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isRunning() {
		return
	}
	l.current.isSuperseded = true
	close(l.current.superseded)
	l.current = nil
}

// getRunning is the request running, or nothing.
func (l *flushCoordinator) getRunning() *flushRequest {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isRunning() {
		return nil
	}
	return l.current
}

// recordFlush records what one page has left.
func (l *flushCoordinator) recordFlush(token string, said flushState) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if p, listening := l.clients[token]; listening && !p.isDisconnected {
		p.state = said
	}
	l.updateProgress()
}

// updateProgress says how far the request has got: everything is written, or a question
// a person has to answer is outstanding. The lock is held.
func (l *flushCoordinator) updateProgress() {
	r := l.current
	if r == nil || r.isSuperseded {
		return
	}
	for _, p := range l.clients {
		if p.state == flushAsking && !r.isQuestionSignaled {
			r.isQuestionSignaled = true
			close(r.questionRaised)
		}
	}
	if r.isAllWrittenSignaled {
		return
	}
	for _, p := range l.clients {
		if p.state != flushWritten {
			return
		}
	}
	r.isAllWrittenSignaled = true
	close(r.allWritten)
}
