// Package vaults answers the vaults service of the editor window: the list of vaults the
// installation holds and the change of the vault the window shows.
package vaults

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/picker"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vaultlist"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Window is the window the list is answered for: the vault it shows now.
type Window interface {
	GetCurrentVault() domain.Vault
}

// Remover takes a vault off the list, and trashes its folder when asked.
type Remover interface {
	Execute(ctx context.Context, id string, showing domain.VaultID, trash bool) error
}

// FolderChooser opens the native folder dialog.
type FolderChooser interface {
	Execute(ctx context.Context, title, startingAt string) (picker.PickedFolder, error)
}

// Dialog answers the request for the machine's native folder dialog, once the application has one.
type Dialog struct {
	chooseFolder atomic.Pointer[FolderChooser]
}

// Handler answers about the vaults this installation holds.
type Handler struct {
	*Dialog
	window Window
	list   *vaultlist.ListVaults
	add    *vaultlist.AddVault
	rename *vaultlist.RenameVault
	open   *vaultlist.OpenVault
	remove Remover
}

// NewHandler is the handler of the vaults service of one window. The folder dialog is set after,
// when the application has one.
func NewHandler(
	current Window, list *vaultlist.ListVaults, add *vaultlist.AddVault, rename *vaultlist.RenameVault,
	open *vaultlist.OpenVault, remove Remover,
) *Handler {
	return &Handler{Dialog: &Dialog{}, window: current, list: list, add: add, rename: rename, open: open, remove: remove}
}

// NewRoute is where the vaults service answers: its path and the handler of its calls. Each call
// goes through counted.
func NewRoute(handler *Handler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewVaultsServiceHandler(handler, counted)
}

// SetFolderChooser sets the native folder dialog a client may ask for.
func (s *Dialog) SetFolderChooser(choose FolderChooser) { s.chooseFolder.Store(&choose) }

// ListVaults returns every vault of the installation. The active vault is
// queried from the window.
func (s *Handler) ListVaults(
	_ context.Context,
	_ *connect.Request[v1.ListVaultsRequest],
) (*connect.Response[v1.ListVaultsResponse], error) {
	target, err := s.list.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListVaultsResponse{Vaults: make([]*v1.Vault, 0, len(target))}
	for _, v := range target {
		out.Vaults = append(out.Vaults, convertVaultToWire(v))
	}
	return connect.NewResponse(out), nil
}

// ChooseFolder opens this machine's native folder dialog.
func (s *Dialog) ChooseFolder(
	ctx context.Context,
	r *connect.Request[v1.ChooseFolderRequest],
) (*connect.Response[v1.ChooseFolderResponse], error) {
	choose := s.chooseFolder.Load()
	if choose == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDialog)
	}
	selected, err := (*choose).Execute(ctx, r.Msg.GetTitle(), r.Msg.GetStartingAt())
	if picker.IsFolderDialogUnavailable(err) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A closed dialog is a valid response with IsChosen false.
	return connect.NewResponse(&v1.ChooseFolderResponse{Path: selected.Path, IsChosen: selected.IsChosen}), nil
}

// errNoDialog is what a build with no native folder dialog answers.
var errNoDialog = errors.New("this build has no folder dialog")

// AddVault turns a folder into a vault on the list. A name another vault has
// gets a number appended, and is not an error here.
func (s *Handler) AddVault(
	_ context.Context,
	r *connect.Request[v1.AddVaultRequest],
) (*connect.Response[v1.AddVaultResponse], error) {
	added, err := s.add.Execute(r.Msg.GetPath(), r.Msg.GetName())
	if err != nil {
		reason, ok := getErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.AddVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.AddVaultResponse{Vault: convertVaultToWire(added)}), nil
}

// RenameVault sets the display name of a vault. The folder name on the
// filesystem is unchanged.
func (s *Handler) RenameVault(
	ctx context.Context,
	r *connect.Request[v1.RenameVaultRequest],
) (*connect.Response[v1.RenameVaultResponse], error) {
	renamed, err := s.rename.Execute(ctx, r.Msg.GetId(), r.Msg.GetName())
	if err != nil {
		reason, ok := getErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RenameVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RenameVaultResponse{Vault: convertVaultToWire(renamed)}), nil
}

// RemoveVault removes a vault from the list and the index, and moves its folder
// to the trash when the request sets the trash flag.
func (s *Handler) RemoveVault(
	ctx context.Context,
	r *connect.Request[v1.RemoveVaultRequest],
) (*connect.Response[v1.RemoveVaultResponse], error) {
	err := s.remove.Execute(ctx, r.Msg.GetId(), s.window.GetCurrentVault().ID, r.Msg.GetTrash())
	if err != nil {
		reason, ok := getErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RemoveVaultResponse{}), nil
}

// OpenVault shows another vault in this window.
func (s *Handler) OpenVault(
	ctx context.Context,
	r *connect.Request[v1.OpenVaultRequest],
) (*connect.Response[v1.OpenVaultResponse], error) {
	err := s.open.Execute(ctx, r.Msg.GetId())
	if err == nil {
		return connect.NewResponse(&v1.OpenVaultResponse{}), nil
	}
	// A closing window or a shutdown in progress stops the operation; the requested
	// vault is unchanged.
	if errors.Is(err, window.ErrClosing) || errors.Is(err, window.ErrFlushing) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.OpenVaultResponse{Error: &reason}), nil
}

func convertVaultToWire(v vaultlist.Listed) *v1.Vault {
	return &v1.Vault{
		Id: string(v.ID), Name: v.Name, Path: v.Path,
		IsMissing: v.IsMissing,
	}
}

func getErrorCode(err error) (v1.VaultsErrorCode, bool) {
	switch {
	case errors.Is(err, vaultdomain.ErrUnreadable):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, vaultdomain.ErrCopy):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY, true
	case errors.Is(err, domain.ErrOverlaps):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS, true
	case errors.Is(err, vaultdomain.ErrNameTaken):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN, true
	case errors.Is(err, vaultdomain.ErrLastVault):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT, true
	case errors.Is(err, vaultdomain.ErrShowing):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_CURRENT, true
	case errors.Is(err, registry.ErrUnknown), errors.Is(err, registry.ErrNoName):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NOT_FOUND, true
	case errors.Is(err, machine.ErrNoTrash):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH, true
	case errors.Is(err, window.ErrAnswerPending):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED, true
	default:
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED, false
	}
}
