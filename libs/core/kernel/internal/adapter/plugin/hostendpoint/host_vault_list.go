package hostendpoint

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
	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
)

var errNoVaultList = errors.New("this host has no list of vaults to change")

func getVaultsErrorCode(err error) (v1.VaultsErrorCode, bool) {
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
	case errors.Is(err, registry.ErrShowing):
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

func (e *vaultEndpoint) beginList(header http.Header) (pluginport.PluginVaultList, error) {
	if _, err := e.endpoint.authorize(header.Get("Authorization"), manifest.PermissionVaultManage); err != nil {
		return nil, err
	}
	list := e.endpoint.callers.GetPluginVaultList()
	if list == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoVaultList)
	}
	return list, nil
}

func (e *vaultEndpoint) findVault(id domain.VaultID) (domain.Vault, error) {
	vault := e.endpoint.callers.GetPluginVault()
	if vault == nil {
		return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), registry.ErrUnknown)
	}
	known, err := vault.GetVaults()
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

func (e *vaultEndpoint) AddVault(
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

func (e *vaultEndpoint) RenameVault(
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

func (e *vaultEndpoint) RemoveVault(
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
func (e *vaultEndpoint) OpenVault(
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
