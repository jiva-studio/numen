// Package vault answers the vault service of the editor window: what the vault in the window is
// doing, and what moved in it.
package vault

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
)

// Window is the window that holds the vault.
type Window interface {
	// GetVaultState is what the window knows about the vault it shows now.
	GetVaultState(ctx context.Context) (window.VaultState, error)
}

// Changes delivers the changes of the vault.
type Changes interface {
	// SubscribeChanges delivers each change of the vault until the returned function is called.
	SubscribeChanges() (<-chan window.Change, func())
}

// Handler answers what the vault in the window is doing: whether it has been scanned, and what
// moved in it.
type Handler struct {
	window  Window
	changes Changes
}

func NewHandler(current Window, changes Changes) *Handler {
	return &Handler{window: current, changes: changes}
}

// NewRoute is where the vault service answers: its path and the handler of its calls. Each call
// goes through counted.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewVaultServiceHandler(handler, counted)
}

// GetVaultState answers the vault the window shows and how far it has been read.
func (h *Handler) GetVaultState(
	ctx context.Context, _ *connect.Request[v1.GetVaultStateRequest],
) (*connect.Response[v1.GetVaultStateResponse], error) {
	state, err := h.window.GetVaultState(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetVaultStateResponse{
		Id:   string(state.Vault.ID),
		Name: state.Vault.Name,
		Path: state.Vault.Path,
		Scan: &v1.Scan{
			IsReady:    state.IsReady,
			Error:      state.Error,
			WatchError: state.WatchError,
		},
		Coverage: &v1.IndexCoverage{Text: state.Coverage},
	}), nil
}

// WatchVaultChanges streams vault changes until the caller disconnects.
func (h *Handler) WatchVaultChanges(
	ctx context.Context,
	_ *connect.Request[v1.WatchVaultChangesRequest],
	out *connect.ServerStream[v1.WatchVaultChangesResponse],
) error {
	line, isDone := h.changes.SubscribeChanges()
	defer isDone()

	// An initial empty message confirms the subscription. A stream that sends
	// nothing until the vault changes is indistinguishable from one that never
	// opened.
	if err := out.Send(&v1.WatchVaultChangesResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.ResendInterval)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := out.Send(&v1.WatchVaultChangesResponse{}); err != nil {
				return err
			}
		case what, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(newChangeResponse(what)); err != nil {
				return err
			}
		}
	}
}

func newChangeResponse(what window.Change) *v1.WatchVaultChangesResponse {
	renamedMoves := make([]*v1.Move, 0, len(what.Renamed))
	for _, went := range what.Renamed {
		renamedMoves = append(renamedMoves, &v1.Move{From: went.From, To: went.To})
	}
	return &v1.WatchVaultChangesResponse{
		Paths: what.Paths, Reload: what.ShouldReload, Renamed: renamedMoves, VersionPaths: what.VersionPaths,
	}
}
