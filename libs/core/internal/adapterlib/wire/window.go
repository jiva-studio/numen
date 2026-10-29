package wire

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// Editor is the name of the editor window. A request names its target window,
// and a window serves only requests naming it.
const Editor = "editor"

// ErrAnotherWindow is a question naming a window other than the one it reached.
var ErrAnotherWindow = errors.New("that is another window")

// A Reason is a human-readable cause for a degraded window state, written by the
// component that detects it and read by the UI.
//
// The empty reason means no problem. It is a string, not a flag, so the UI can
// display the cause.
type Reason struct{ why atomic.Value }

// Store records the reason, or the empty string when there is no problem.
func (r *Reason) Store(why string) { r.why.Store(why) }

// Why returns the last stored reason, or the empty string when there is no problem.
func (r *Reason) Why() string {
	why, _ := r.why.Load().(string)
	return why
}

// Window is one window of this application, as the schema describes it: the
// running tasks and the connected clients that take part in the flush at quit.
//
// Both windows share this implementation, so tasks have one wire shape and the
// quit drain is implemented once.
type Window struct {
	// Named is the window this answers for, which a question names to reach it.
	Named string

	// Tasking is the set of running tasks. A nil value reports an empty list.
	Tasking *run.Tasks

	// Vault is the identity of the vault this window has in front of the
	// person. Nil is a window open on the installation, which answers with
	// none.
	Vault func() string

	clients flushCoordinator
}

// Again is the interval at which a stream resends its last message while nothing
// has changed.
//
// The request context belongs to the process and is cancelled when the window
// closes, not when a page reloads, so a handler detects a gone client only
// through a failed write. Every stream therefore writes periodically.
const Again = time.Second

// WatchTasks streams the running tasks until the client disconnects.
//
// The full list is sent on every change, and the first message is sent
// immediately: a client that connects while work is running needs the current
// state, and a stream silent until the next change cannot be told from one that
// never opened.
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

// WatchQuit streams quit notifications until the client disconnects.
func (w *Window) WatchQuit(
	ctx context.Context,
	r *connect.Request[v1.WatchQuitRequest],
	out *connect.ServerStream[v1.WatchQuitResponse],
) error {
	if err := w.checkWindowName(r.Msg.GetWindow()); err != nil {
		return err
	}
	token, notified, isDone := w.clients.registerClient()
	defer isDone()

	// The token is sent first: a client without one cannot answer a flush request,
	// and a stream silent until quit cannot be told from one that never opened.
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
			// The same token is resent. A disconnected page fails the write, which ends
			// this stream and releases the quit.
			if err := out.Send(&v1.WatchQuitResponse{Token: token}); err != nil {
				return err
			}
		case request, isOpen := <-notified:
			if !isOpen {
				return nil
			}
			if err := out.Send(&v1.WatchQuitResponse{Token: request, Flush: true}); err != nil {
				return err
			}
		}
	}
}

// ReportFlush records the flush state a client reports.
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

// GetCurrentVault returns the identity of the vault open in this window.
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

// checkWindowName returns an error unless the request names this window.
func (w *Window) checkWindowName(named string) error {
	if named == w.Named {
		return nil
	}
	return connect.NewError(connect.CodeNotFound, ErrAnotherWindow)
}

// AddTask puts one piece of work in the list.
func (w *Window) AddTask(pos run.Task) {
	if w != nil && w.Tasking != nil {
		w.Tasking.Set(pos)
	}
}

// RemoveTask takes one piece of work out of the list.
func (w *Window) RemoveTask(id string) {
	if w != nil && w.Tasking != nil {
		w.Tasking.Remove(id)
	}
}

// RequestFlush asks every client to write its unsaved work and waits for the
// request to end. It returns false when a client holds work awaiting a user
// decision, or when another request superseded this one.
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

// WaitForFlush waits for the running request to end with every client written.
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

// CancelFlush ends the running request and releases its waiters.
func (w *Window) CancelFlush() { w.clients.endRequest() }

// describeTasks converts tasks to their schema form.
func describeTasks(list []run.Task) []*v1.Task {
	out := make([]*v1.Task, 0, len(list))
	for _, item := range list {
		out = append(out, &v1.Task{
			Id:            item.ID,
			Title:         item.Title,
			About:         item.About,
			Done:          item.Count,
			Total:         item.Total,
			Error:         item.Error,
			IsUserStarted: item.IsUserStarted,
			Unit:          convertUnit(item.Unit),
			PluginId:      item.PluginID,
			PluginName:    item.PluginName,
		})
	}
	return out
}

// convertUnit converts a task unit to its schema form.
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
	// flushUnanswered is a client that has not responded to this request.
	flushUnanswered flushState = iota
	// flushWritten is a client with no unsaved work left.
	flushWritten
	// flushAsking is a client holding unwritten work while the user decides what to
	// do with it.
	flushAsking
)

// parseFlushResult converts a client's flush result to the internal flush state.
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

// flushCoordinator tracks the connected clients and the running flush request.
//
// A client holds unwritten work in a buffer this process cannot read, so it is
// asked to write and responds when done. A disconnected client is not awaited,
// except when its stream ended in the flushAsking state: its work is still
// unwritten, so it counts as unanswered until a client reconnects.
type flushCoordinator struct {
	mu      sync.Mutex
	next    int
	clients map[string]*client
	current *flushRequest
}

// client is one browser tab connected to the window, with its flush state.
type client struct {
	flushTokens chan string
	state       flushState
	// isDisconnected marks a client whose stream ended in the flushAsking state.
	// It receives no tokens and records no state; a newly connected client replaces it.
	isDisconnected bool
}

// flushRequest is one flush request to all clients, with its progress.
type flushRequest struct {
	// allWritten closes once every client has written its work.
	allWritten chan struct{}
	// questionRaised closes once a client holds work awaiting a user decision.
	questionRaised chan struct{}
	// superseded closes once another request replaces this one.
	superseded chan struct{}

	// The flush coordinator's lock guards these fields.
	isAllWrittenSignaled bool
	isQuestionSignaled   bool
	isSuperseded         bool
}

// isPending reports whether a client is awaiting a user decision.
func (r *flushRequest) isPending() bool {
	select {
	case <-r.questionRaised:
		return true
	default:
		return false
	}
}

// registerClient registers a client and returns its flush token, the channel on
// which it receives flush tokens, and a function that unregisters it.
func (l *flushCoordinator) registerClient() (string, <-chan string, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clients == nil {
		l.clients = map[string]*client{}
	}
	// One window shows one vault, so a connecting client replaces the disconnected
	// client and takes over its pending work.
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

// disconnectClient unregisters a client, or marks it disconnected if it is flushAsking.
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

// startRequest begins a request: every client is told to write, and states from
// the previous request are cleared. A running request is superseded and its
// waiters are released.
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

// endRequest ends the running request and releases its waiters.
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
func (l *flushCoordinator) recordFlush(token string, response flushState) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if p, found := l.clients[token]; found && !p.isDisconnected {
		p.state = response
	}
	l.updateProgress()
}

// updateProgress signals allWritten or questionRaised on the running request
// when the client states warrant it. The lock is held.
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
