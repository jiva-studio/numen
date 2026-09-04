package webui

import (
	"context"
	"errors"
	"os"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// vaultsService answers about the vaults this installation holds, over the API
// this window serves.
type vaultsService struct{ api *API }

// errNoVaults is what a build that holds no list of vaults answers.
var errNoVaults = errors.New("this build holds no list of vaults")

// errNoChanging is what a build that cannot change that list answers.
var errNoChanging = errors.New("this build cannot change the vaults this installation holds")

// errNoFolderDialog is what a build with no window to put a folder dialog in
// front of answers.
var errNoFolderDialog = errors.New("this build has no folder picker")

// errNoOpening is what a build that cannot move the window to another vault
// answers.
var errNoOpening = errors.New("this build cannot show another vault")

// ListVaults is every vault the installation holds. Which of them the window
// has in front of the person is asked of the window.
func (s vaultsService) ListVaults(
	_ context.Context,
	_ *connect.Request[v1.ListVaultsRequest],
) (*connect.Response[v1.ListVaultsResponse], error) {
	if s.api.Vaults.Registry == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	held, err := usecase.List{Registry: s.api.Vaults.Registry}.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListVaultsResponse{Vaults: make([]*v1.Known, 0, len(held))}
	for _, v := range held {
		out.Vaults = append(out.Vaults, knownOf(v))
	}
	return connect.NewResponse(out), nil
}

// ChooseFolder puts this machine's own folder dialog in front of the person.
func (s vaultsService) ChooseFolder(
	ctx context.Context,
	r *connect.Request[v1.ChooseFolderRequest],
) (*connect.Response[v1.ChooseFolderResponse], error) {
	if s.api.Vaults.FolderDialog == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoFolderDialog)
	}
	path, chose, err := s.api.Vaults.FolderDialog.Choose(ctx, r.Msg.GetTitle(), r.Msg.GetStartingAt())
	if errors.Is(err, port.ErrChoosing) || errors.Is(err, port.ErrNoFolderDialog) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A person who closed the dialog chose nothing, and that is an answer.
	return connect.NewResponse(&v1.ChooseFolderResponse{Path: path, Chose: chose}), nil
}

// AddVault turns a folder into a vault on the list. A name another vault has
// gets a number appended, and is not a refusal here.
func (s vaultsService) AddVault(
	_ context.Context,
	r *connect.Request[v1.AddVaultRequest],
) (*connect.Response[v1.AddVaultResponse], error) {
	if s.api.Vaults.Add == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	added, err := s.api.Vaults.Add.Execute(r.Msg.GetPath(), r.Msg.GetDisplayName())
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.AddVaultResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.AddVaultResponse{Vault: knownOf(added)}), nil
}

// RenameVault is what a person calls a vault. The folder keeps the name the
// filesystem gives it.
func (s vaultsService) RenameVault(
	ctx context.Context,
	r *connect.Request[v1.RenameVaultRequest],
) (*connect.Response[v1.RenameVaultResponse], error) {
	if s.api.Vaults.Registry == nil || s.api.Vaults.Rename == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	v, err := s.found(r.Msg.GetName())
	if err == nil {
		v, err = s.api.Vaults.Rename.Execute(ctx, v, r.Msg.GetDisplayName())
	}
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RenameVaultResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.RenameVaultResponse{Vault: knownOf(v)}), nil
}

// ForgetVault takes a vault off the list and out of the index. The folder stays
// where it is.
func (s vaultsService) ForgetVault(
	ctx context.Context,
	r *connect.Request[v1.ForgetVaultRequest],
) (*connect.Response[v1.ForgetVaultResponse], error) {
	if s.api.Vaults.Registry == nil || s.api.Vaults.Forget == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	v, err := s.offTheList(r.Msg.GetName())
	if err == nil {
		err = s.api.Vaults.Forget.Execute(ctx, v)
	}
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ForgetVaultResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.ForgetVaultResponse{}), nil
}

// EraseVault is ForgetVault, and the folder goes to the place this machine
// keeps what a person deleted.
func (s vaultsService) EraseVault(
	ctx context.Context,
	r *connect.Request[v1.EraseVaultRequest],
) (*connect.Response[v1.EraseVaultResponse], error) {
	if s.api.Vaults.Registry == nil || s.api.Vaults.Erase == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	v, err := s.offTheList(r.Msg.GetName())
	if err == nil {
		err = s.api.Vaults.Erase.Execute(ctx, v)
	}
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.EraseVaultResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.EraseVaultResponse{}), nil
}

// OpenVault shows another vault in this window.
func (s vaultsService) OpenVault(
	ctx context.Context,
	r *connect.Request[v1.OpenVaultRequest],
) (*connect.Response[v1.OpenVaultResponse], error) {
	if s.api.Vaults.Registry == nil || s.api.Opens == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoOpening)
	}
	v, err := s.found(r.Msg.GetName())
	if err == nil {
		err = s.api.Opens(ctx, v)
	}
	if err == nil {
		return connect.NewResponse(&v1.OpenVaultResponse{}), nil
	}
	// A window that is closing, or already settling what it owes, is what
	// stopped this, and the vault asked for is as it was.
	if errors.Is(err, errGoing) || errors.Is(err, errSettling) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	refusal, refused := vaultRefusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.OpenVaultResponse{Refusal: &refusal}), nil
}

// offTheList is the vault an identity names, asked to leave. The vault the
// window has in front of the person stays: the use cases are not told which one
// that is, and this is.
func (s vaultsService) offTheList(id string) (domain.Vault, error) {
	v, err := s.found(id)
	if err != nil {
		return domain.Vault{}, err
	}
	if v.ID == s.api.Showing().ID {
		return domain.Vault{}, errShowing
	}
	return v, nil
}

// errShowing is the vault the window has in front of the person, asked to go.
var errShowing = errors.New("this vault is the one the window is showing")

// found is the vault an identity names.
func (s vaultsService) found(id string) (domain.Vault, error) {
	return usecase.Find{Registry: s.api.Vaults.Registry}.Execute(id)
}

// knownOf is one vault as the schema carries it. A folder that is not there to
// be found is marked, and the vault stays on the list.
func knownOf(v domain.Vault) *v1.Known {
	_, err := os.Stat(v.Path)
	return &v1.Known{
		Name: string(v.ID), DisplayName: v.Name, Path: v.Path, Missing: err != nil,
	}
}

// vaultRefusedBy says which refusal an error about the list is, and whether it
// is one at all. Anything else is the list, the index or the machine being out
// of reach.
func vaultRefusedBy(err error) (v1.VaultsRefusal, bool) {
	switch {
	case errors.Is(err, usecase.ErrUnreadable):
		return v1.VaultsRefusal_VAULTS_REFUSAL_UNREADABLE, true
	case errors.Is(err, usecase.ErrCopy):
		return v1.VaultsRefusal_VAULTS_REFUSAL_COPY, true
	case errors.Is(err, domain.ErrOverlaps):
		return v1.VaultsRefusal_VAULTS_REFUSAL_OVERLAPS, true
	case errors.Is(err, usecase.ErrNameTaken):
		return v1.VaultsRefusal_VAULTS_REFUSAL_NAME_TAKEN, true
	case errors.Is(err, usecase.ErrLastVault):
		return v1.VaultsRefusal_VAULTS_REFUSAL_LAST_VAULT, true
	case errors.Is(err, errShowing):
		return v1.VaultsRefusal_VAULTS_REFUSAL_SHOWING, true
	case errors.Is(err, usecase.ErrUnknown):
		return v1.VaultsRefusal_VAULTS_REFUSAL_UNKNOWN, true
	case errors.Is(err, port.ErrNoTrash):
		return v1.VaultsRefusal_VAULTS_REFUSAL_NO_TRASH, true
	case errors.Is(err, errAsking):
		return v1.VaultsRefusal_VAULTS_REFUSAL_ASKING, true
	default:
		return v1.VaultsRefusal_VAULTS_REFUSAL_UNSPECIFIED, false
	}
}
