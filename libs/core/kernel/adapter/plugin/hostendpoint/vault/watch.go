package vault

import (
	"context"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (e *watchCalls) WatchMoves(
	ctx context.Context, req *connect.Request[v1.WatchMovesRequest], stream *connect.ServerStream[v1.WatchMovesResponse],
) error {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return err
	}
	if vault == nil {
		return connect.NewError(connect.CodeFailedPrecondition, plugin.ErrNoVault)
	}
	moves, err := vault.Watch.WatchMoves(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if err := stream.Send(&v1.WatchMovesResponse{}); err != nil {
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
			if err := stream.Send(&v1.WatchMovesResponse{}); err != nil {
				return err
			}
		}
	}
}

func (e *watchCalls) WatchVault(
	ctx context.Context, req *connect.Request[v1.WatchVaultRequest], stream *connect.ServerStream[v1.WatchVaultResponse],
) error {
	_, vault, err := e.begin(req.Header(), manifest.PermissionVaultRead)
	if err != nil {
		return err
	}
	if vault == nil {
		return connect.NewError(connect.CodeFailedPrecondition, plugin.ErrNoVault)
	}
	watched, showing := vault.Switch.GetVault()
	if !showing {
		return connect.NewError(connect.CodeFailedPrecondition, plugin.ErrNoVault)
	}
	changes, missing, err := vault.Watch.Watch(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if err := stream.Send(&v1.WatchVaultResponse{}); err != nil {
		return err
	}
	return followVault(ctx, vault, watched, changes, missing, stream)
}

func followVault(
	ctx context.Context, vault *Access, watched domain.Vault,
	changes <-chan []string, lostChan <-chan struct{}, stream *connect.ServerStream[v1.WatchVaultResponse],
) error {
	poll := time.NewTicker(vaultWatchPoll)
	defer poll.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-lostChan:
			return nil
		case paths, open := <-changes:
			if !open {
				return nil
			}
			if err := stream.Send(&v1.WatchVaultResponse{Paths: paths}); err != nil {
				return err
			}
		case <-poll.C:
			if now, showing := vault.Switch.GetVault(); !showing || now.ID != watched.ID {
				return stream.Send(&v1.WatchVaultResponse{IsVaultChanged: true})
			}
		}
	}
}
