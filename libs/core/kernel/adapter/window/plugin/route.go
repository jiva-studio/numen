// Package plugin serves the route on which the windows reach the services plugins offer, and the
// plugins that run.
package plugin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// Processes lists the plugins that run at once and after each change, until the context ends.
type Processes interface {
	Execute(ctx context.Context) <-chan plugindomain.ProcessList
}

// Release stops the process of a plugin that has finished its work.
type Release interface {
	Execute(ctx context.Context, pluginID string)
}

// Handler answers PluginService to a window. The window does not manage plugins, so GetPlugins is
// unimplemented.
type Handler struct {
	numenv1connect.UnimplementedPluginServiceHandler
	processes Processes
	release   Release
}

var _ numenv1connect.PluginServiceHandler = (*Handler)(nil)

var errNoPluginID = errors.New("the request names no plugin")

// NewHandler answers over the plugins that run, and releases their processes through release.
func NewHandler(processes Processes, release Release) *Handler {
	return &Handler{processes: processes, release: release}
}

// NewRoute is where PluginService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewRoute(handler *Handler, options connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewPluginServiceHandler(handler, options)
	return path, served, handler
}

// WatchPluginProcesses sends the plugins that run at once, on every change and again every
// wire.ResendInterval, until the use case closes its channel or a write fails. A closed client is
// detected only by the failed write.
func (h *Handler) WatchPluginProcesses(
	ctx context.Context, _ *connect.Request[v1.WatchPluginProcessesRequest], out *connect.ServerStream[v1.WatchPluginProcessesResponse],
) error {
	watching, stop := context.WithCancel(ctx)
	defer stop()
	watch := h.processes.Execute(watching)
	repeat := time.NewTicker(wire.ResendInterval)
	defer repeat.Stop()

	var last plugindomain.ProcessList
	for {
		select {
		case list, isOpen := <-watch:
			if !isOpen {
				return nil
			}
			if err := out.Send(describe(list)); err != nil {
				return err
			}
			last = list
			repeat.Reset(wire.ResendInterval)
		case <-repeat.C:
			if err := out.Send(describe(last)); err != nil {
				return err
			}
		}
	}
}

func describe(list plugindomain.ProcessList) *v1.WatchPluginProcessesResponse {
	return &v1.WatchPluginProcessesResponse{Processes: wire.DescribePluginProcesses(list.Processes), TimeUnixMs: list.Time.UnixMilli()}
}

// ReleaseProcess stops the process of the plugin the request names once it holds no lease. A request
// that names no plugin answers InvalidArgument.
func (h *Handler) ReleaseProcess(
	ctx context.Context, req *connect.Request[v1.ReleaseProcessRequest],
) (*connect.Response[v1.ReleaseProcessResponse], error) {
	pluginID := req.Msg.GetPluginId()
	if pluginID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoPluginID)
	}
	h.release.Execute(ctx, pluginID)
	return connect.NewResponse(&v1.ReleaseProcessResponse{}), nil
}
