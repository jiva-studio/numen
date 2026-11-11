package vault

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

var errNoVaultList = errors.New("this host has no list of vaults to change")

func getVaultsErrorCode(err error) (v1.VaultsErrorCode, bool) {
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
	case errors.Is(err, plugin.ErrUnsavedChanges):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSAVED, true
	default:
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED, false
	}
}

func getVaultsResult(err error) (code v1.VaultsErrorCode, detail string, callErr error) {
	if err == nil {
		return 0, "", nil
	}
	if code, ok := getVaultsErrorCode(err); ok {
		return code, err.Error(), nil
	}
	if errors.Is(err, domain.ErrUnavailable) {
		return 0, "", connect.NewError(connect.CodeUnavailable, err)
	}
	return 0, "", connect.NewError(connect.CodeInternal, err)
}

func (e *vaultsCalls) beginList(header http.Header) (List, error) {
	if _, err := e.auth.Authorize(header.Get("Authorization"), manifest.PermissionVaultManage); err != nil {
		return nil, err
	}
	list := e.source.GetPluginVaultList()
	if list == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoVaultList)
	}
	return list, nil
}

func (e *vaultsCalls) findVault(id domain.VaultID) (domain.Vault, error) {
	vault := e.source.GetPluginVault()
	if vault == nil {
		return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), registry.ErrUnknown)
	}
	known, err := vault.Switch.GetVaults()
	if err != nil {
		return domain.Vault{}, err
	}
	for _, one := range known {
		if one.ID == id {
			return one, nil
		}
	}
	return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), registry.ErrUnknown)
}

func writeVault(v domain.Vault) *v1.Vault {
	return &v1.Vault{Id: string(v.ID), Name: v.Name, Path: v.Path}
}

func (e *vaultsCalls) AddVault(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceAddVaultRequest],
) (*connect.Response[v1.HostVaultServiceAddVaultResponse], error) {
	list, err := e.beginList(req.Header())
	if err != nil {
		return nil, err
	}
	vault, addErr := list.AddVault(ctx, req.Msg.GetRoot(), req.Msg.GetName())
	code, detail, err := getVaultsResult(addErr)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Vault: writeVault(vault)}), nil
}

func (e *vaultsCalls) RenameVault(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRenameVaultRequest],
) (*connect.Response[v1.HostVaultServiceRenameVaultResponse], error) {
	list, err := e.beginList(req.Header())
	if err != nil {
		return nil, err
	}
	vault, err := list.RenameVault(ctx, domain.VaultID(req.Msg.GetId()), req.Msg.GetName())
	code, detail, err := getVaultsResult(err)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Vault: writeVault(vault)}), nil
}

func (e *vaultsCalls) RemoveVault(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveVaultRequest],
) (*connect.Response[v1.HostVaultServiceRemoveVaultResponse], error) {
	list, err := e.beginList(req.Header())
	if err != nil {
		return nil, err
	}
	id := domain.VaultID(req.Msg.GetId())
	// The folder is read from the list before the vault leaves it.
	entry, err := e.findVault(id)
	if err == nil {
		err = list.RemoveVault(ctx, id)
	}
	code, detail, err := getVaultsResult(err)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Path: entry.Path}), nil
}

// OpenVault returns once the window shows the vault or has rejected the open. A rejection for unsaved
// changes returns UNSAVED with the window's message in detail.
func (e *vaultsCalls) OpenVault(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceOpenVaultRequest],
) (*connect.Response[v1.HostVaultServiceOpenVaultResponse], error) {
	list, err := e.beginList(req.Header())
	if err != nil {
		return nil, err
	}
	id := domain.VaultID(req.Msg.GetId())
	_, openErr := e.findVault(id)
	if openErr == nil {
		openErr = list.OpenVault(ctx, id)
	}
	code, detail, err := getVaultsResult(openErr)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{}), nil
}
