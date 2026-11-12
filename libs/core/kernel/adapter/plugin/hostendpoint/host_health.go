package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

var errNoHealth = errors.New("the host keeps no health items for plugins")

// healthEndpoint answers the calls of PluginHealthService a plugin makes about itself. The plugin is the
// one the token names; a plugin_id in the request is not read. Listing is the windows' and is not served.
type healthEndpoint struct {
	numenv1connect.UnimplementedPluginHealthServiceHandler

	endpoint *Endpoint
}

func (e *healthEndpoint) ReportHealth(
	_ context.Context, req *connect.Request[v1.ReportHealthRequest],
) (*connect.Response[v1.ReportHealthResponse], error) {
	pluginID, err := e.endpoint.authenticate(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	report := e.endpoint.callers.Health.Report
	if report == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoHealth)
	}
	if err := report.Execute(pluginID, wire.ReadHealthItem(req.Msg.GetItem())); err != nil {
		return nil, getHealthError(err)
	}
	return connect.NewResponse(&v1.ReportHealthResponse{}), nil
}

func (e *healthEndpoint) ClearHealth(
	_ context.Context, req *connect.Request[v1.ClearHealthRequest],
) (*connect.Response[v1.ClearHealthResponse], error) {
	pluginID, err := e.endpoint.authenticate(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	clearing := e.endpoint.callers.Health.Clear
	if clearing == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoHealth)
	}
	if err := clearing.Execute(pluginID, req.Msg.GetKey()); err != nil {
		return nil, getHealthError(err)
	}
	return connect.NewResponse(&v1.ClearHealthResponse{}), nil
}

func getHealthError(err error) error {
	if errors.Is(err, plugin.ErrInvalidHealth) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
