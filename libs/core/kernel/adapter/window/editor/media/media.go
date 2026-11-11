// Package media serves a window's MediaService: where the bytes of a file of the shown vault, or of
// a plugin's store of it, are loaded from.
package media

import (
	"context"
	"errors"
	"io/fs"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Vault names the vault the window shows. A window showing none returns a vault with no ID.
type Vault interface {
	GetCurrentVault() domain.Vault
}

// Addresses gives the loopback address a file is loaded from, or nothing while the window has no
// socket that answers.
type Addresses interface {
	GetMediaAddress(v domain.Vault, stat domain.FileStat) string
	GetPluginFileAddress(v domain.Vault, address file.Address, size int64) string
}

var (
	errNoVault  = errors.New("the window has no vault")
	errComingUp = errors.New("the vault is still coming up")
)

// Handler serves MediaService.
type Handler struct {
	vault          Vault
	addresses      Addresses
	resolveAddress *file.ResolveAddress
}

// NewHandler is the handler of MediaService over the use cases that find a file.
func NewHandler(window Vault, addresses Addresses, resolve *file.ResolveAddress) *Handler {
	return &Handler{vault: window, addresses: addresses, resolveAddress: resolve}
}

// ResolveAddress returns the loopback address of the bytes of the file at the address and the media
// type for a player.
func (h *Handler) ResolveAddress(
	ctx context.Context, r *connect.Request[v1.ResolveAddressRequest],
) (*connect.Response[v1.ResolveAddressResponse], error) {
	showing := h.vault.GetCurrentVault()
	if showing.ID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	if h.resolveAddress == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	resolved, err := h.resolveAddress.Execute(ctx, showing, r.Msg.GetAddress())
	switch {
	case errors.Is(err, file.ErrInvalidAddress):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, file.ErrNoPluginFiles):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, connect.NewError(getListCode(err), err)
	}
	src := h.addresses.GetMediaAddress(showing, resolved.Stat)
	if resolved.Address.IsPluginFile() {
		src = h.addresses.GetPluginFileAddress(showing, resolved.Address, resolved.Size)
	}
	return connect.NewResponse(&v1.ResolveAddressResponse{
		Src:       src,
		MediaType: resolved.MediaType,
	}), nil
}

func getListCode(err error) connect.Code {
	switch {
	case errors.Is(err, vault.ErrOutside):
		return connect.CodeInvalidArgument
	case errors.Is(err, fs.ErrNotExist):
		return connect.CodeNotFound
	default:
		return connect.CodeInternal
	}
}
