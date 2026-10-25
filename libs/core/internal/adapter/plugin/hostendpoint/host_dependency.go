package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// dependencyStreamBuffer is how many statuses a streaming caller may lag behind.
const dependencyStreamBuffer = 32

var errNoDependencies = errors.New("the host provisions no dependencies")

// dependencyEndpoint answers DependencyService for the plugins that call the host.
type dependencyEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.DependencyServiceHandler = (*dependencyEndpoint)(nil)

// begin authenticates the call and returns what provisions the dependencies.
func (e *dependencyEndpoint) begin(header http.Header) (pluginport.DependencyCatalog, error) {
	if _, err := e.endpoint.authenticate(header.Get("Authorization")); err != nil {
		return nil, err
	}
	catalog := e.endpoint.callers.GetDependencies()
	if catalog == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoDependencies)
	}
	return catalog, nil
}

func (e *dependencyEndpoint) GetDependencyStatus(
	ctx context.Context, req *connect.Request[v1.GetDependencyStatusRequest],
) (*connect.Response[v1.GetDependencyStatusResponse], error) {
	catalog, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	status, err := catalog.GetStatus(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetDependencyStatusResponse{Status: writeWireDependencyStatus(status)}), nil
}

func (e *dependencyEndpoint) ListDependencies(
	ctx context.Context, req *connect.Request[v1.ListDependenciesRequest],
) (*connect.Response[v1.ListDependenciesResponse], error) {
	catalog, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	statuses, err := catalog.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.DependencyStatus, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, writeWireDependencyStatus(status))
	}
	return connect.NewResponse(&v1.ListDependenciesResponse{Statuses: out}), nil
}

// ProvisionDependency streams the status of the dependency until it is ready or has failed.
func (e *dependencyEndpoint) ProvisionDependency(
	ctx context.Context, req *connect.Request[v1.ProvisionDependencyRequest], stream *connect.ServerStream[v1.ProvisionDependencyResponse],
) error {
	catalog, err := e.begin(req.Header())
	if err != nil {
		return err
	}
	requirement, err := readWireDependencyRequirement(req.Msg.GetRequirement())
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !catalog.CanProvision(requirement) {
		return connect.NewError(connect.CodeNotFound, pluginport.ErrNoProvisioner)
	}
	updates := make(chan pluginport.DependencyStatus, dependencyStreamBuffer)
	errc := make(chan error, 1)
	go func() {
		_, err := catalog.Provision(ctx, requirement, func(status pluginport.DependencyStatus) { sendLatest(updates, status) })
		errc <- err
	}()
	for {
		select {
		case status := <-updates:
			if err := stream.Send(&v1.ProvisionDependencyResponse{Status: writeWireDependencyStatus(status)}); err != nil {
				return err
			}
		case err := <-errc:
			return e.finishStream(stream, updates, requirement, err)
		}
	}
}

// finishStream sends what is left of the statuses and, for a run that failed
// without reporting it, the failure.
func (e *dependencyEndpoint) finishStream(
	stream *connect.ServerStream[v1.ProvisionDependencyResponse], updates chan pluginport.DependencyStatus, req pluginport.DependencyRequirement, runErr error,
) error {
	var last *pluginport.DependencyStatus
	for drained := false; !drained; {
		select {
		case status := <-updates:
			if err := stream.Send(&v1.ProvisionDependencyResponse{Status: writeWireDependencyStatus(status)}); err != nil {
				return err
			}
			last = &status
		default:
			drained = true
		}
	}
	if runErr != nil && (last == nil || last.State != pluginport.DependencyStateFailed) {
		failed := pluginport.DependencyStatus{ID: req.ID, State: pluginport.DependencyStateFailed, Error: runErr.Error()}
		return stream.Send(&v1.ProvisionDependencyResponse{Status: writeWireDependencyStatus(failed)})
	}
	return nil
}

// sendLatest puts the status on the channel. A caller that lags is told the newest
// status and not every one.
func sendLatest(updates chan pluginport.DependencyStatus, status pluginport.DependencyStatus) {
	for {
		select {
		case updates <- status:
			return
		default:
		}
		select {
		case <-updates:
		default:
		}
	}
}

func readWireDependencyRequirement(wire *v1.DependencyRequirement) (pluginport.DependencyRequirement, error) {
	kind, err := readWireDependencyKind(wire.GetKind())
	if err != nil {
		return pluginport.DependencyRequirement{}, err
	}
	return pluginport.DependencyRequirement{
		ID: wire.GetId(), Kind: kind, Version: wire.GetVersion(), SourceURL: wire.GetSourceUrl(),
		ChecksumSHA256: wire.GetChecksumSha256(), Metadata: wire.GetMetadata(),
	}, nil
}

func readWireDependencyKind(wire v1.DependencyKind) (pluginport.DependencyKind, error) {
	switch wire {
	case v1.DependencyKind_DEPENDENCY_KIND_RUNTIME:
		return pluginport.DependencyKindRuntime, nil
	case v1.DependencyKind_DEPENDENCY_KIND_MODEL:
		return pluginport.DependencyKindModel, nil
	case v1.DependencyKind_DEPENDENCY_KIND_BINARY:
		return pluginport.DependencyKindBinary, nil
	case v1.DependencyKind_DEPENDENCY_KIND_DATASET:
		return pluginport.DependencyKindDataset, nil
	case v1.DependencyKind_DEPENDENCY_KIND_UNSPECIFIED:
		return "", pluginport.ErrInvalidDependency
	}
	return "", pluginport.ErrInvalidDependency
}

func writeWireDependencyStatus(status pluginport.DependencyStatus) *v1.DependencyStatus {
	wire := &v1.DependencyStatus{
		Id: status.ID, State: writeWireDependencyState(status.State), BytesDownloaded: status.BytesDownloaded,
		BytesTotal: status.BytesTotal, ProgressPercent: status.ProgressPercent, Message: status.Message, Error: status.Error,
	}
	if status.Resolved != nil {
		wire.Resolved = &v1.ResolvedDependency{
			Id: status.Resolved.ID, RootPath: status.Resolved.RootPath, PrimaryPath: status.Resolved.PrimaryPath,
			EnvVars: status.Resolved.EnvVars, Metadata: status.Resolved.Metadata,
		}
	}
	return wire
}

func writeWireDependencyState(state pluginport.DependencyState) v1.DependencyState {
	switch state {
	case pluginport.DependencyStateMissing:
		return v1.DependencyState_DEPENDENCY_STATE_MISSING
	case pluginport.DependencyStateProvisioning:
		return v1.DependencyState_DEPENDENCY_STATE_PROVISIONING
	case pluginport.DependencyStateReady:
		return v1.DependencyState_DEPENDENCY_STATE_READY
	case pluginport.DependencyStateFailed:
		return v1.DependencyState_DEPENDENCY_STATE_FAILED
	}
	return v1.DependencyState_DEPENDENCY_STATE_UNSPECIFIED
}
