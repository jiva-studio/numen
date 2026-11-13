// Package plugin serves the route on which the windows reach the services plugins offer, and the
// plugins that run.
package plugin

import (
	"context"
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
	Execute(ctx context.Context) <-chan []plugindomain.ProcessStatus
}

// Handler answers PluginService to a window. The window does not manage plugins, so GetPlugins is
// unimplemented.
type Handler struct {
	numenv1connect.UnimplementedPluginServiceHandler
	processes Processes
}

var _ numenv1connect.PluginServiceHandler = (*Handler)(nil)

// NewHandler answers over the plugins that run.
func NewHandler(processes Processes) *Handler {
	return &Handler{processes: processes}
}

// NewRoute is where PluginService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewRoute(handler *Handler, options connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewPluginServiceHandler(handler, options)
	return path, served, handler
}

// WatchPluginProcesses sends the plugins that run at once, on every change and again every
// wire.ResendInterval, until the client is gone. A gone client is noticed through a failed write.
func (h *Handler) WatchPluginProcesses(
	ctx context.Context, _ *connect.Request[v1.WatchPluginProcessesRequest], out *connect.ServerStream[v1.WatchPluginProcessesResponse],
) error {
	watching, stop := context.WithCancel(ctx)
	defer stop()
	watch := h.processes.Execute(watching)
	repeat := time.NewTicker(wire.ResendInterval)
	defer repeat.Stop()

	var last []plugindomain.ProcessStatus
	for {
		select {
		case list, isOpen := <-watch:
			if !isOpen {
				return nil
			}
			if err := out.Send(&v1.WatchPluginProcessesResponse{Processes: wire.DescribePluginProcesses(list)}); err != nil {
				return err
			}
			last = list
			repeat.Reset(wire.ResendInterval)
		case <-repeat.C:
			if err := out.Send(&v1.WatchPluginProcessesResponse{Processes: wire.DescribePluginProcesses(last)}); err != nil {
				return err
			}
		}
	}
}
