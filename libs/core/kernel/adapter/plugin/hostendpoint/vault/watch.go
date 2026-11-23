package vault

import (
	"context"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (e *watchCalls) WatchIndexChanges(
	ctx context.Context, req *connect.Request[v1.WatchIndexChangesRequest], stream *connect.ServerStream[v1.WatchIndexChangesResponse],
) error {
	vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return err
	}
	if vault == nil {
		return connect.NewError(connect.CodeFailedPrecondition, plugin.ErrNoVault)
	}
	moves, err := vault.Watch.WatchIndexChanges(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if err := stream.Send(&v1.WatchIndexChangesResponse{}); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, open := <-moves:
			if !open {
				return nil
			}
			if err := stream.Send(&v1.WatchIndexChangesResponse{}); err != nil {
				return err
			}
		}
	}
}
