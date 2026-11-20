package hostendpoint

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

var errNoProcesses = errors.New("the host does not stop plugins on request")

// processEndpoint answers the calls of PluginService a plugin makes about its own process. The plugin is
// the one the token names. Listing is the windows' and is not served.
type processEndpoint struct {
	numenv1connect.UnimplementedPluginServiceHandler

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

const (
	maxModelStages = 16
	maxStageBytes  = 96
)

func (e *processEndpoint) ReportModelProviders(_ context.Context, req *connect.Request[v1.ReportModelProvidersRequest]) (*connect.Response[v1.ReportModelProvidersResponse], error) {
	header := req.Header().Get("Authorization")
	id, err := e.endpoint.Authenticate(header)
	if err != nil {
		return nil, err
	}
	if e.endpoint.callers.ModelProviders == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoProcesses)
	}
	providers := req.Msg.GetProviders()
	if req.Msg.GetRevision() == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing model provider revision"))
	}
	if len(providers) > maxModelStages {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("too many model stages"))
	}
	seen := make(map[plugin.ModelProvider]bool, len(providers))
	rows := make([]plugin.ModelProvider, 0, len(providers))
	for _, row := range providers {
		stage, provider := row.GetStage(), row.GetProvider()
		if stage == "" || provider == "" || len(stage) > maxStageBytes || len(provider) > maxStageBytes {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid model stage or provider"))
		}
		entry := plugin.ModelProvider{Stage: stage, Provider: provider}
		if seen[entry] {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("duplicate model stage and provider"))
		}
		seen[entry] = true
		rows = append(rows, entry)
	}
	if !e.endpoint.callers.ModelProviders(id, strings.TrimPrefix(header, bearerPrefix), req.Msg.GetRevision(), rows) {
		return nil, connect.NewError(connect.CodeUnauthenticated, errUnknownCaller)
	}
	return connect.NewResponse(&v1.ReportModelProvidersResponse{}), nil
}
