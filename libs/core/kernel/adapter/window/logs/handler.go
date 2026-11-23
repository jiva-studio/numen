// Package logs serves LogService to the windows: streaming structured log entries and recording
// entries reported by a window or a plugin frontend.
package logs

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// Watcher streams the recent and incoming log entries until ctx ends.
type Watcher interface {
	Execute(ctx context.Context) <-chan []domainlogs.Entry
}

// Recorder validates and appends a structured log entry.
type Recorder interface {
	Execute(entry domainlogs.Entry) error
}

// Handler answers LogService to a window.
type Handler struct {
	watcher  Watcher
	recorder Recorder
	getLevel func() domainlogs.Level
	setLevel func(domainlogs.Level) error
}

var _ numenv1connect.LogServiceHandler = (*Handler)(nil)

func NewHandler(
	watcher Watcher,
	recorder Recorder,
	getLevel func() domainlogs.Level,
	setLevel func(domainlogs.Level) error,
) *Handler {
	return &Handler{watcher: watcher, recorder: recorder, getLevel: getLevel, setLevel: setLevel}
}

func (h *Handler) GetLogLevel(_ context.Context, _ *connect.Request[v1.GetLogLevelRequest]) (*connect.Response[v1.GetLogLevelResponse], error) {
	level := h.getLevel()
	return connect.NewResponse(&v1.GetLogLevelResponse{Level: wire.DescribeLogLevel(level)}), nil
}

func (h *Handler) SetLogLevel(_ context.Context, req *connect.Request[v1.SetLogLevelRequest]) (*connect.Response[v1.SetLogLevelResponse], error) {
	if err := h.setLevel(wire.ReadLogLevel(req.Msg.GetLevel())); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.SetLogLevelResponse{}), nil
}

// NewRoute returns the path and the HTTP handler of LogService.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewLogServiceHandler(handler, counted)
}

// WriteLog appends an entry reported by a window or a plugin frontend.
func (h *Handler) WriteLog(
	_ context.Context, req *connect.Request[v1.WriteLogRequest],
) (*connect.Response[v1.WriteLogResponse], error) {
	entry := wire.ReadLogEntry(req.Msg)
	if entry.Source == "" {
		entry.Source = domainlogs.SourceWindow
	}
	if err := h.recorder.Execute(entry); err != nil {
		if errors.Is(err, domainlogs.ErrEmptyMessage) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteLogResponse{}), nil
}

// WatchLogs sends the recent log entries at once, each new batch as entries arrive, and a heartbeat
// every wire.ResendInterval until the caller disconnects.
func (h *Handler) WatchLogs(
	ctx context.Context, _ *connect.Request[v1.WatchLogsRequest], out *connect.ServerStream[v1.WatchLogsResponse],
) error {
	return wire.RunWatch(ctx, out, wire.ResendInterval, nil, func(ctx context.Context, send func(*v1.WatchLogsResponse) error) error {
		watch := h.watcher.Execute(ctx)
		for {
			select {
			case <-ctx.Done():
				return nil
			case batch, isOpen := <-watch:
				if !isOpen {
					return nil
				}
				if err := send(&v1.WatchLogsResponse{Entries: wire.DescribeLogEntries(batch)}); err != nil {
					return err
				}
			}
		}
	})
}
