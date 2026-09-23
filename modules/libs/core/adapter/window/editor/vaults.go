package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// vaultsService answers about the vaults this installation holds, over the API
// this window serves.
type vaultsService struct{ api *API }

// ListVaults is every vault the installation holds. Which of them the window
// has in front of the person is asked of the window.
func (s vaultsService) ListVaults(
	_ context.Context,
	_ *connect.Request[v1.ListVaultsRequest],
) (*connect.Response[v1.ListVaultsResponse], error) {
	held, err := vaults.NewList(s.api.Vaults.Registry).Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListVaultsResponse{Vaults: make([]*v1.Vault, 0, len(held))}
	for _, v := range held {
		out.Vaults = append(out.Vaults, vaultOf(v, s.api.Readers))
	}
	return connect.NewResponse(out), nil
}

// ChooseFolder puts this machine's own folder dialog in front of the person.
func (s vaultsService) ChooseFolder(
	ctx context.Context,
	r *connect.Request[v1.ChooseFolderRequest],
) (*connect.Response[v1.ChooseFolderResponse], error) {
	path, chose, err := s.api.Vaults.FolderDialog.Choose(ctx, r.Msg.GetTitle(), r.Msg.GetStartingAt())
	if errors.Is(err, port.ErrChoosing) || errors.Is(err, port.ErrNoFolderDialog) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A person who closed the dialog chose nothing, and that is an answer.
	return connect.NewResponse(&v1.ChooseFolderResponse{Path: path, IsChosen: chose}), nil
}

// AddVault turns a folder into a vault on the list. A name another vault has
// gets a number appended, and is not an error here.
func (s vaultsService) AddVault(
	_ context.Context,
	r *connect.Request[v1.AddVaultRequest],
) (*connect.Response[v1.AddVaultResponse], error) {
	added, err := s.api.Vaults.Add.Execute(r.Msg.GetPath(), r.Msg.GetName())
	if err != nil {
		reason, refused := vaultsErrorCodeBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.AddVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.AddVaultResponse{Vault: vaultOf(added, s.api.Readers)}), nil
}

// RenameVault is what a person calls a vault. The folder keeps the name the
// filesystem gives it.
func (s vaultsService) RenameVault(
	ctx context.Context,
	r *connect.Request[v1.RenameVaultRequest],
) (*connect.Response[v1.RenameVaultResponse], error) {
	v, err := s.found(r.Msg.GetId())
	if err == nil {
		v, err = s.api.Vaults.Rename.Execute(ctx, v, r.Msg.GetName())
	}
	if err != nil {
		reason, refused := vaultsErrorCodeBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RenameVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RenameVaultResponse{Vault: vaultOf(v, s.api.Readers)}), nil
}

// RemoveVault takes a vault off the list and out of the index, and its folder
// to the trash when the request asks for it.
func (s vaultsService) RemoveVault(
	ctx context.Context,
	r *connect.Request[v1.RemoveVaultRequest],
) (*connect.Response[v1.RemoveVaultResponse], error) {
	v, err := s.offTheList(r.Msg.GetId())
	if err == nil {
		err = s.removal(ctx, v, r.Msg.GetTrash())
	}
	if err != nil {
		reason, refused := vaultsErrorCodeBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RemoveVaultResponse{}), nil
}

// removal is the two ways a vault leaves the list.
func (s vaultsService) removal(ctx context.Context, v domain.Vault, trash bool) error {
	if trash {
		_, err := s.api.Vaults.Erase.Execute(ctx, v)
		return err
	}
	return s.api.Vaults.Forget.Execute(ctx, v)
}

// OpenVault shows another vault in this window.
func (s vaultsService) OpenVault(
	ctx context.Context,
	r *connect.Request[v1.OpenVaultRequest],
) (*connect.Response[v1.OpenVaultResponse], error) {
	v, err := s.found(r.Msg.GetId())
	if err == nil {
		err = s.api.Opens(ctx, v)
	}
	if err == nil {
		return connect.NewResponse(&v1.OpenVaultResponse{}), nil
	}
	// A window that is closing, or already settling what it owes, is what
	// stopped this, and the vault asked for is as it was.
	if errors.Is(err, errWindowClosing) || errors.Is(err, errSettling) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	reason, refused := vaultsErrorCodeBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.OpenVaultResponse{Error: &reason}), nil
}

// offTheList is the vault an identity names, asked to leave. The vault the
// window has in front of the person stays: the use cases are not told which one
// that is, and this is.
func (s vaultsService) offTheList(id string) (domain.Vault, error) {
	v, err := s.found(id)
	if err != nil {
		return domain.Vault{}, err
	}
	if v.ID == s.api.GetShownVault().ID {
		return domain.Vault{}, errShowing
	}
	return v, nil
}

// errShowing is the vault the window has in front of the person, asked to go.
var errShowing = errors.New("this vault is the one the window is showing")

// found is the vault an identity names.
func (s vaultsService) found(id string) (domain.Vault, error) {
	return vaults.NewFind(s.api.Vaults.Registry).Execute(id)
}

// vaultOf is one vault as the schema carries it. A folder that is not there to
// be found is marked, and the vault stays on the list.
func vaultOf(v domain.Vault, readers port.VaultReaders) *v1.Vault {
	return &v1.Vault{
		Id: string(v.ID), Name: v.Name, Path: v.Path,
		IsMissing: vaults.NewFolderCheck(readers).Execute(v),
	}
}

// vaultsErrorCodeBy says which code an error about the list carries, and
// whether it carries one at all. Anything else is the list, the index or the
// machine being out of reach.
func vaultsErrorCodeBy(err error) (v1.VaultsErrorCode, bool) {
	switch {
	case errors.Is(err, vaults.ErrUnreadable):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, vaults.ErrCopy):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY, true
	case errors.Is(err, domain.ErrOverlaps):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS, true
	case errors.Is(err, vaults.ErrNameTaken):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN, true
	case errors.Is(err, vaults.ErrLastVault):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT, true
	case errors.Is(err, errShowing):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING, true
	case errors.Is(err, vaults.ErrUnknown):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN, true
	case errors.Is(err, port.ErrNoTrash):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH, true
	case errors.Is(err, errAsking):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING, true
	default:
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED, false
	}
}
