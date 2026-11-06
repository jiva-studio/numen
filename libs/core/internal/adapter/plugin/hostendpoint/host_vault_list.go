package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
)

var errNoVaultList = errors.New("this host has no list of vaults to change")

// getVaultsFailure is the error code of a change of the list of vaults that was refused, and false
// for any other failure.
func getVaultsFailure(err error) (v1.VaultsErrorCode, bool) {
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
	case errors.Is(err, vaults.ErrShowing):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING, true
	case errors.Is(err, vaults.ErrUnknown), errors.Is(err, vaults.ErrNoName):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN, true
	case errors.Is(err, pluginport.ErrVaultAsking):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING, true
	default:
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED, false
	}
}

// answerVaults is the failure a change of the list answers with, or the error of the call.
func answerVaults(err error) (*v1.VaultsErrorCode, error) {
	if err == nil {
		return nil, nil
	}
	if failure, ok := getVaultsFailure(err); ok {
		return &failure, nil
	}
	if errors.Is(err, pluginport.ErrUnavailable) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return nil, connect.NewError(connect.CodeInternal, err)
}

// beginList authorizes a change of the list of vaults and returns the list. A host with no list
// answers that it is unavailable.
func (e *vaultEndpoint) beginList(header http.Header) (pluginport.PluginVaultList, error) {
	if _, err := e.endpoint.authorize(header.Get("Authorization"), pluginport.PermissionVaultManage); err != nil {
		return nil, err
	}
	list := e.endpoint.callers.GetPluginVaultList()
	if list == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errNoVaultList)
	}
	return list, nil
}

// findVault is the vault of the identity on the list the plugins read.
func (e *vaultEndpoint) findVault(id domain.VaultID) (domain.Vault, error) {
	vault := e.endpoint.callers.GetPluginVault()
	if vault == nil {
		return domain.Vault{}, fmt.Errorf("vault %q: %w", id, vaults.ErrUnknown)
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
	return domain.Vault{}, fmt.Errorf("vault %q: %w", id, vaults.ErrUnknown)
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
	failure, err := answerVaults(addErr)
	if err != nil {
		return nil, err
	}
	if failure != nil {
		return connect.NewResponse(&v1.HostVaultServiceAddVaultResponse{Failure: failure, Detail: addErr.Error()}), nil
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
	failure, err := answerVaults(err)
	if err != nil || failure != nil {
		return connect.NewResponse(&v1.HostVaultServiceRenameVaultResponse{Failure: failure}), err
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
	failure, err := answerVaults(err)
	if err != nil || failure != nil {
		return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Failure: failure}), err
	}
	return connect.NewResponse(&v1.HostVaultServiceRemoveVaultResponse{Path: entry.Path}), nil
}

// OpenVault answers once the window has decided, before it changes vault: the change closes the
// session of the calling plugin, so it runs under a context the end of the call does not cancel.
func (e *vaultEndpoint) OpenVault(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceOpenVaultRequest],
) (*connect.Response[v1.HostVaultServiceOpenVaultResponse], error) {
	list, err := e.beginList(req.Header())
	if err != nil {
		return nil, err
	}
	id := domain.VaultID(req.Msg.GetId())
	if _, err := e.findVault(id); err != nil {
		failure, err := answerVaults(err)
		return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{Failure: failure}), err
	}
	swap, err := list.OpenVault(ctx, id)
	if err != nil {
		failure, err := answerVaults(err)
		return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{Failure: failure}), err
	}
	isStarted := e.endpoint.background.start(ctx, func(ctx context.Context) {
		if err := swap(ctx); err != nil {
			e.endpoint.background.report(fmt.Errorf("open vault %s for a plugin: %w", id, err))
		}
	})
	if !isStarted {
		return nil, connect.NewError(connect.CodeUnavailable, errEndpointShutdown)
	}
	return connect.NewResponse(&v1.HostVaultServiceOpenVaultResponse{}), nil
}
