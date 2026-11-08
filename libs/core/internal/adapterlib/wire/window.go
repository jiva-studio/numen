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

// ErrNoCancellableTask is a cancel naming a task that is not listed or cannot
// be canceled.
var ErrNoCancellableTask = errors.New("no cancellable task has that id")

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
	why, isText := r.why.Load().(string)
	if !isText {
		return ""
	}
	return why
}

// Window is one window of this application, as the schema describes it: the
// running tasks and the connected clients that take part in the flush at quit.
//
// Both windows share this implementation, so tasks have one wire shape and the
// quit drain is implemented once.
type Window struct {
	Named string

	Tasking *run.Tasks

	Vault func() string

	clients flushCoordinator
}

// Again is the interval at which a stream resends its last message while nothing
// has changed.
// The request context is canceled when the window closes, not when a page
// reloads, so a handler detects a gone client only through a failed write.
const Again = time.Second

// WatchTasks streams the running tasks until the client disconnects.
// The full list is sent on every change, and the first message is sent
// immediately: a stream silent until the next change is indistinguishable from
// one that never opened.
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

	// The token is sent first: a client without one cannot respond to a flush
	// request, and a stream silent until quit is indistinguishable from one
	// that never opened.
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

// CancelTask stops the listed task with the id the request names.
func (w *Window) CancelTask(
	_ context.Context,
	r *connect.Request[v1.CancelTaskRequest],
) (*connect.Response[v1.CancelTaskResponse], error) {
	if err := w.checkWindowName(r.Msg.GetWindow()); err != nil {
		return nil, err
	}
	if w.Tasking == nil || !w.Tasking.Cancel(r.Msg.GetId()) {
		return nil, connect.NewError(connect.CodeNotFound, ErrNoCancellableTask)
	}
	return connect.NewResponse(&v1.CancelTaskResponse{}), nil
}

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

// RequestFlush requests every client to write its unsaved work and waits for
// the request to end. It returns false when a client holds work awaiting a user
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

// IsQuestionOpen reports whether the running request has a client awaiting a user decision with
// its work unwritten.
func (w *Window) IsQuestionOpen() bool {
	request := w.clients.getRunning()
	return request != nil && request.isPending() && !request.isWritten()
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
			IsCancellable: item.Cancel != nil,
			Kind:          convertKind(item.Kind),
		})
	}
	return out
}

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
	// flushUnsaved is a client with unwritten work while the user decides what to
	// do with it.
	flushUnsaved
)

func parseFlushResult(result v1.FlushResult) flushState {
	switch result {
	case v1.FlushResult_FLUSH_RESULT_NOTHING, v1.FlushResult_FLUSH_RESULT_WRITTEN:
		return flushWritten
	case v1.FlushResult_FLUSH_RESULT_UNSAVED:
		return flushUnsaved
	default:
		return flushUnanswered
	}
}

// flushCoordinator tracks the connected clients and the running flush request.
//
// A client holds unwritten work in a buffer this process cannot read, so it
// receives a write request and responds when done. A disconnected client is not
// awaited, except when its stream ended in the flushUnsaved state: its work is
// still unwritten, so it counts as unanswered until a client reconnects.
type flushCoordinator struct {
	mu      sync.Mutex
	next    int
	clients map[string]*client
	current *flushRequest
}

// client is one browser tab connected to the window, with its flush state.
type client struct {
	flushTokens    chan string
	state          flushState
	isDisconnected bool
}

// flushRequest is one flush request to all clients, with its progress.
type flushRequest struct {
	allWritten     chan struct{}
	questionRaised chan struct{}
	superseded     chan struct{}

	isAllWrittenSignaled bool
	isQuestionSignaled   bool
	isSuperseded         bool
}

func (r *flushRequest) isPending() bool {
	select {
	case <-r.questionRaised:
		return true
	default:
		return false
	}
}

func (r *flushRequest) isWritten() bool {
	select {
	case <-r.allWritten:
		return true
	default:
		return false
	}
}

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

func (l *flushCoordinator) isRunning() bool {
	return l.current != nil && !l.current.isSuperseded
}

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
	if p.state == flushUnsaved {
		p.state = flushUnanswered
		p.isDisconnected = true
	} else {
		delete(l.clients, token)
	}
	l.updateProgress()
}

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

func (l *flushCoordinator) getRunning() *flushRequest {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isRunning() {
		return nil
	}
	return l.current
}

func (l *flushCoordinator) recordFlush(token string, response flushState) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if p, found := l.clients[token]; found && !p.isDisconnected {
		p.state = response
	}
	l.updateProgress()
}

func (l *flushCoordinator) updateProgress() {
	r := l.current
	if r == nil || r.isSuperseded {
		return
	}
	for _, p := range l.clients {
		if p.state == flushUnsaved && !r.isQuestionSignaled {
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

func convertKind(in run.Kind) v1.TaskKind {
	if in == run.KindNotice {
		return v1.TaskKind_TASK_KIND_NOTICE
	}
	return v1.TaskKind_TASK_KIND_UNSPECIFIED
}
