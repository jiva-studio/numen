package wire

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
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
	Name string

	Tasks TaskList

	Vault func() string

	clients flushCoordinator
}

// ResendInterval is the interval at which a stream resends its last message while nothing
// has changed.
// The request context is canceled when the window closes, not when a page
// reloads, so a handler detects a gone client only through a failed write.
const ResendInterval = time.Second

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
	if w.Tasks == nil {
		// A window that does nothing behind itself still answers, so a client
		// has one thing to listen to.
		return out.Send(&v1.WatchTasksResponse{})
	}

	watching, stop := context.WithCancel(ctx)
	defer stop()

	watch := w.Tasks.Watch(watching)
	repeat := time.NewTicker(ResendInterval)
	defer repeat.Stop()

	var last []work.Task
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
			repeat.Reset(ResendInterval)

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

	repeat := time.NewTicker(ResendInterval)
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
	if w.Tasks == nil || !w.Tasks.Cancel(r.Msg.GetId()) {
		return nil, connect.NewError(connect.CodeNotFound, ErrNoCancellableTask)
	}
	return connect.NewResponse(&v1.CancelTaskResponse{}), nil
}

func (w *Window) checkWindowName(named string) error {
	if named == w.Name {
		return nil
	}
	return connect.NewError(connect.CodeNotFound, ErrAnotherWindow)
}

// AddTask puts one piece of work in the list.
func (w *Window) AddTask(pos work.Task) {
	if w != nil && w.Tasks != nil {
		w.Tasks.Set(pos)
	}
}

// RemoveTask takes one piece of work out of the list.
func (w *Window) RemoveTask(id string) {
	if w != nil && w.Tasks != nil {
		w.Tasks.Remove(id)
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
	return !request.isPending() && w.clients.getRunningRequest() == request
}

// IsQuestionOpen reports whether the running request has a client awaiting a user decision with
// its work unwritten.
func (w *Window) IsQuestionOpen() bool {
	request := w.clients.getRunningRequest()
	return request != nil && request.isPending() && !request.isWritten()
}

// WaitForFlush waits for the running request to end with every client written.
func (w *Window) WaitForFlush(ctx context.Context) bool {
	request := w.clients.getRunningRequest()
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

func describeTasks(list []work.Task) []*v1.Task {
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

func convertUnit(in work.Unit) v1.Unit {
	switch in {
	case work.Bytes:
		return v1.Unit_UNIT_BYTES
	case work.Seconds:
		return v1.Unit_UNIT_SECONDS
	case work.Things:
		return v1.Unit_UNIT_THINGS
	}
	return v1.Unit_UNIT_THINGS
}

func convertKind(in work.Kind) v1.TaskKind {
	if in == work.KindNotice {
		return v1.TaskKind_TASK_KIND_NOTICE
	}
	return v1.TaskKind_TASK_KIND_UNSPECIFIED
}

// TaskList is the list of what the window is doing, which the window shows and lets the user cancel from.
type TaskList interface {
	Watch(ctx context.Context) <-chan []work.Task
	Cancel(id string) bool
	Set(task work.Task)
	Remove(id string)
	List() []work.Task
}
