// Package plugin serves the route on which the windows reach the services plugins offer, and the
// processes of the plugins.
package plugin

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// Processes lists the plugins that run at once and after each change, until the context ends.
type Processes interface {
	Execute(ctx context.Context) <-chan plugindomain.ProcessList
}

// Handler answers PluginProcessService to a window.
type Handler struct {
	processes Processes
	retry     *setting.RetryPlugin
}

var _ numenv1connect.PluginProcessServiceHandler = (*Handler)(nil)

// NewHandler answers over the plugins that run, and starts a plugin that failed to start through retry.
func NewHandler(processes Processes, retry *setting.RetryPlugin) *Handler {
	return &Handler{processes: processes, retry: retry}
}

// NewRoute is where PluginProcessService is mounted, and what the schema generated to answer it.
func NewRoute(handler *Handler, options connect.HandlerOption) (path string, served http.Handler) {
	path, served = numenv1connect.NewPluginProcessServiceHandler(handler, options)
	return path, served
}

// errNoRetry is what a build that cannot start a plugin again answers.
var errNoRetry = errors.New("this build does not start plugins again")

// RetryPlugin starts a plugin whose last start failed once more. A plugin that fails
// again is answered with its reason, and one the host does not know is not found.
func (h *Handler) RetryPlugin(
	ctx context.Context, r *connect.Request[v1.RetryPluginRequest],
) (*connect.Response[v1.RetryPluginResponse], error) {
	if h.retry == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoRetry)
	}
	failure, err := h.retry.Execute(ctx, r.Msg.GetPluginId())
	if err != nil {
		var unavailable *plugindomain.UnavailableError
		if errors.As(err, &unavailable) && unavailable.Reason == plugindomain.NotInstalled {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, wire.NewInternalError(err)
	}
	reply := &v1.RetryPluginResponse{}
	if failure != nil {
		reply.Failure = failure.Reason
	}
	return connect.NewResponse(reply), nil
}

// WatchPluginProcesses sends the plugins that run at once, on every change and again every
// wire.ResendInterval, until the use case closes its channel or a write fails. A closed client is
// detected only by the failed write.
func (h *Handler) WatchPluginProcesses(
	ctx context.Context, _ *connect.Request[v1.WatchPluginProcessesRequest], out *connect.ServerStream[v1.WatchPluginProcessesResponse],
) error {
	return wire.RunWatch(ctx, out, wire.ResendInterval, wire.GetLastOrEmpty, func(ctx context.Context, send func(*v1.WatchPluginProcessesResponse) error) error {
		watch := h.processes.Execute(ctx)
		for {
			select {
			case <-ctx.Done():
				return nil
			case list, isOpen := <-watch:
				if !isOpen {
					return nil
				}
				if err := send(describe(list)); err != nil {
					return err
				}
			}
		}
	})
}

func describe(list plugindomain.ProcessList) *v1.WatchPluginProcessesResponse {
	return &v1.WatchPluginProcessesResponse{Processes: wire.DescribePluginProcesses(list.Processes), TimeUnixMs: list.Time.UnixMilli()}
}
