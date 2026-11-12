package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// dependencyStreamBuffer is how many statuses a streaming caller may lag behind.
const dependencyStreamBuffer = 32

// Dependencies is what the host provisions the dependencies of the plugins with, and answers their
// questions about them.
type Dependencies interface {
	GetStatus(ctx context.Context, id string) (plugin.DependencyStatus, error)
	List(ctx context.Context) ([]plugin.DependencyStatus, error)
	CanProvision(req manifest.DependencyRequirement) bool
	Provision(ctx context.Context, req manifest.DependencyRequirement, onProgress func(plugin.DependencyStatus)) (plugin.ResolvedDependency, error)
}

var errNoDependencies = errors.New("the host provisions no dependencies")

// dependencyEndpoint answers DependencyService for the plugins that call the host.
type dependencyEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.DependencyServiceHandler = (*dependencyEndpoint)(nil)

func (e *dependencyEndpoint) begin(header http.Header) (Dependencies, error) {
	if _, err := e.endpoint.authenticate(header.Get("Authorization")); err != nil {
		return nil, err
	}
	catalog := e.endpoint.callers.Dependencies()
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
		return connect.NewError(connect.CodeNotFound, plugin.ErrNoProvisioner)
	}
	updates := make(chan plugin.DependencyStatus, dependencyStreamBuffer)
	errc := make(chan error, 1)
	go func() {
		_, err := catalog.Provision(ctx, requirement, func(status plugin.DependencyStatus) { sendLatest(updates, status) })
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

func (e *dependencyEndpoint) finishStream(
	stream *connect.ServerStream[v1.ProvisionDependencyResponse], updates chan plugin.DependencyStatus, req manifest.DependencyRequirement, runErr error,
) error {
	var last *plugin.DependencyStatus
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
	if runErr != nil && (last == nil || last.State != plugin.DependencyStateFailed) {
		failed := plugin.DependencyStatus{ID: req.ID, State: plugin.DependencyStateFailed, Error: runErr.Error()}
		return stream.Send(&v1.ProvisionDependencyResponse{Status: writeWireDependencyStatus(failed)})
	}
	return nil
}

func sendLatest(updates chan plugin.DependencyStatus, status plugin.DependencyStatus) {
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

func readWireDependencyRequirement(wire *v1.DependencyRequirement) (manifest.DependencyRequirement, error) {
	kind, err := readWireDependencyKind(wire.GetKind())
	if err != nil {
		return manifest.DependencyRequirement{}, err
	}
	return manifest.DependencyRequirement{
		ID: wire.GetId(), Kind: kind, Version: wire.GetVersion(), SourceURL: wire.GetSourceUrl(),
		ChecksumSHA256: wire.GetChecksumSha256(), Metadata: wire.GetMetadata(),
	}, nil
}

func readWireDependencyKind(wire v1.DependencyKind) (manifest.DependencyKind, error) {
	switch wire {
	case v1.DependencyKind_DEPENDENCY_KIND_RUNTIME:
		return manifest.DependencyKindRuntime, nil
	case v1.DependencyKind_DEPENDENCY_KIND_MODEL:
		return manifest.DependencyKindModel, nil
	case v1.DependencyKind_DEPENDENCY_KIND_BINARY:
		return manifest.DependencyKindBinary, nil
	case v1.DependencyKind_DEPENDENCY_KIND_DATASET:
		return manifest.DependencyKindDataset, nil
	case v1.DependencyKind_DEPENDENCY_KIND_UNSPECIFIED:
		return "", manifest.ErrInvalidDependency
	}
	return "", manifest.ErrInvalidDependency
}

func writeWireDependencyStatus(status plugin.DependencyStatus) *v1.DependencyStatus {
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

func writeWireDependencyState(state plugin.DependencyState) v1.DependencyState {
	switch state {
	case plugin.DependencyStateMissing:
		return v1.DependencyState_DEPENDENCY_STATE_MISSING
	case plugin.DependencyStateProvisioning:
		return v1.DependencyState_DEPENDENCY_STATE_PROVISIONING
	case plugin.DependencyStateReady:
		return v1.DependencyState_DEPENDENCY_STATE_READY
	case plugin.DependencyStateFailed:
		return v1.DependencyState_DEPENDENCY_STATE_FAILED
	}
	return v1.DependencyState_DEPENDENCY_STATE_UNSPECIFIED
}
