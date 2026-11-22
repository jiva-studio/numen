package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var errNoProcesses = errors.New("the host does not stop plugins on request")

// processEndpoint answers the calls of HostProcessService a plugin makes about its own process. The plugin is
// the one the token names.
type processEndpoint struct {
	endpoint *Endpoint
}

func (e *processEndpoint) ReleaseProcess(
	ctx context.Context, req *connect.Request[v1.ReleaseProcessRequest],
) (*connect.Response[v1.ReleaseProcessResponse], error) {
	pluginID, err := e.endpoint.Authenticate(req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	processes := e.endpoint.callers.Processes
	if processes == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoProcesses)
	}
	// The stop ends the process that waits for this answer, which cancels ctx.
	processes.ReleaseProcess(context.WithoutCancel(ctx), pluginID)
	return connect.NewResponse(&v1.ReleaseProcessResponse{}), nil
}
