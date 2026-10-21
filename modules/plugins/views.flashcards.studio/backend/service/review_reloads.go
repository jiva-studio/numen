package service

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// again is how often a stream that has seen nothing move says so, so that a page can tell a
// stream that is open from one that is not.
const again = time.Second

// WatchReloads says something moved, for as long as the caller listens.
//
// What moved is not carried: what this window shows is counts and the cards behind them, and they are
// asked for again whatever changed.
func (a *reviewHandler) WatchReloads(
	ctx context.Context,
	_ *connect.Request[v1.WatchReloadsRequest],
	out *connect.ServerStream[v1.WatchReloadsResponse],
) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	moved, err := a.vault.WatchMoves(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnavailable, err)
	}

	// Named as listening before anything has moved. A stream that says nothing until a file changes
	// cannot be told from one that never opened.
	if err := out.Send(&v1.WatchReloadsResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// Nothing moved, which is what the opening message says too. A page that has gone fails the
			// write.
			if err := out.Send(&v1.WatchReloadsResponse{}); err != nil {
				return err
			}
		case _, open := <-moved:
			if !open {
				return nil
			}
			a.scenarios().flash.CardFaces.InvalidateAll()
			if err := out.Send(&v1.WatchReloadsResponse{Reload: true}); err != nil {
				return err
			}
		}
	}
}
