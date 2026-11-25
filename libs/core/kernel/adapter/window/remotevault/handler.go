// Package remotevault serves a window's RemoteVaultService: the vaults a plugin keeps remotely, listed next
// to the local vaults, and the call that opens one in a new local folder.
package remotevault

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Handler answers RemoteVaultService.
type Handler struct {
	list    *vault.ListRemoteVaults
	suggest *vault.SuggestVaultPath
	open    *vault.OpenRemoteVault
}

var _ numenv1connect.RemoteVaultServiceHandler = (*Handler)(nil)

// NewHandler is the handler of RemoteVaultService over its use cases.
func NewHandler(list *vault.ListRemoteVaults, suggest *vault.SuggestVaultPath, open *vault.OpenRemoteVault) *Handler {
	return &Handler{list: list, suggest: suggest, open: open}
}

// NewRoute is where the remote vault service answers: its path and the handler of its calls. Each call goes
// through counted.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewRemoteVaultServiceHandler(handler, counted)
}

// ListRemoteVaults lists the remote vaults the provider offers, and none when it offers none.
func (h *Handler) ListRemoteVaults(
	ctx context.Context, _ *connect.Request[v1.RemoteVaultServiceListRemoteVaultsRequest],
) (*connect.Response[v1.RemoteVaultServiceListRemoteVaultsResponse], error) {
	listed := h.list.Execute(ctx)
	out := &v1.RemoteVaultServiceListRemoteVaultsResponse{Vaults: make([]*v1.RemoteVault, 0, len(listed))}
	for _, one := range listed {
		out.Vaults = append(out.Vaults, &v1.RemoteVault{
			RemoteId: one.RemoteID, Name: one.Name, SizeBytes: one.SizeBytes, VaultId: string(one.VaultID),
		})
	}
	return connect.NewResponse(out), nil
}

// SuggestVaultPath names the folder of a new vault of the name.
func (h *Handler) SuggestVaultPath(
	_ context.Context, r *connect.Request[v1.SuggestVaultPathRequest],
) (*connect.Response[v1.SuggestVaultPathResponse], error) {
	suggested, err := h.suggest.Execute(r.Msg.GetName(), r.Msg.GetFolder())
	if err != nil {
		return nil, newOpenError(err)
	}
	return connect.NewResponse(&v1.SuggestVaultPathResponse{
		Folder: suggested.Folder, Path: suggested.Path, IsTaken: suggested.IsTaken,
	}), nil
}

// OpenRemoteVault makes the folder at the path, registers it as a vault and connects the remote vault to it.
func (h *Handler) OpenRemoteVault(
	ctx context.Context, r *connect.Request[v1.OpenRemoteVaultRequest],
) (*connect.Response[v1.OpenRemoteVaultResponse], error) {
	if r.Msg.GetRemoteId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the remote vault is not named"))
	}
	id, err := h.open.Execute(ctx, r.Msg.GetRemoteId(), r.Msg.GetPath())
	if err != nil {
		return nil, newOpenError(err)
	}
	return connect.NewResponse(&v1.OpenRemoteVaultResponse{VaultId: string(id)}), nil
}

func newOpenError(err error) error {
	var answered *connect.Error
	if errors.As(err, &answered) {
		return answered
	}
	var unavailable *plugindomain.UnavailableError
	switch {
	case errors.As(err, &unavailable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, vaultdomain.ErrInvalidPath), errors.Is(err, domain.ErrOverlaps),
		errors.Is(err, vaultdomain.ErrUnreadable), errors.Is(err, vaultdomain.ErrCopy):
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return wire.NewInternalError(err)
}
