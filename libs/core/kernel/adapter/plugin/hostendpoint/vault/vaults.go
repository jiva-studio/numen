package vault

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

var (
	errNoVaultList      = errors.New("this host has no list of vaults to change")
	errNoTrash          = errors.New("a plugin removes a vault from the list and leaves its folder")
	errManagerNotCaller = errors.New("a vault is managed by the plugin that adds it")
	errNotManager       = errors.New("a plugin removes only the vaults it manages")
)

func getVaultsErrorCode(err error) (v1.ErrorCode, bool) {
	switch {
	case errors.Is(err, vaultdomain.ErrUnreadable):
		return v1.ErrorCode_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, vaultdomain.ErrCopy):
		return v1.ErrorCode_ERROR_CODE_VAULT_COPY, true
	case errors.Is(err, domain.ErrOverlaps):
		return v1.ErrorCode_ERROR_CODE_VAULT_OVERLAPS, true
	case errors.Is(err, vaultdomain.ErrNameTaken):
		return v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS, true
	case errors.Is(err, vaultdomain.ErrLastVault):
		return v1.ErrorCode_ERROR_CODE_LAST_VAULT, true
	case errors.Is(err, vaultdomain.ErrShowing):
		return v1.ErrorCode_ERROR_CODE_VAULT_CURRENT, true
	case errors.Is(err, vaultdomain.ErrUnknown), errors.Is(err, vaultdomain.ErrNoName):
		return v1.ErrorCode_ERROR_CODE_NOT_FOUND, true
	case errors.Is(err, plugin.ErrUnsavedChanges):
		return v1.ErrorCode_ERROR_CODE_UNSAVED, true
	default:
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, false
	}
}

func getVaultsResult(err error) (code v1.ErrorCode, detail string, callErr error) {
	if err == nil {
		return 0, "", nil
	}
	if code, ok := getVaultsErrorCode(err); ok {
		return code, err.Error(), nil
	}
	if errors.Is(err, domain.ErrUnavailable) {
		return 0, "", connect.NewError(connect.CodeUnavailable, err)
	}
	if errors.Is(err, vaultdomain.ErrManagedExisting) {
		return 0, "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	return 0, "", connect.NewError(connect.CodeInternal, err)
}

func (e *vaultsCalls) beginList(header http.Header) (List, error) {
	_, list, err := e.beginListAs(header)
	return list, err
}

func (e *vaultsCalls) beginListAs(header http.Header) (string, List, error) {
	pluginID, err := e.auth.Authenticate(header.Get("Authorization"))
	if err != nil {
		return "", nil, err
	}
	list := e.source.GetPluginVaultList()
	if list == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errNoVaultList)
	}
	return pluginID, list, nil
}

func (e *vaultsCalls) findVault(id domain.VaultID) (domain.Vault, error) {
	vault := e.source.GetPluginVault()
	if vault == nil {
		return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), vaultdomain.ErrUnknown)
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
	return domain.Vault{}, domain.NewDetailError(fmt.Sprintf("the vault %s is not on the list", id), vaultdomain.ErrUnknown)
}

func writeVault(v domain.Vault) *v1.Vault {
	return &v1.Vault{Id: string(v.ID), Name: v.Name, Path: v.Path, ManagedBy: v.ManagedBy}
}

// ListVaults lists every vault of the installation. A host with no vault lends lists none.
func (e *vaultsCalls) ListVaults(
	_ context.Context, req *connect.Request[v1.ListVaultsRequest],
) (*connect.Response[v1.ListVaultsResponse], error) {
	vault, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	if vault == nil {
		return connect.NewResponse(&v1.ListVaultsResponse{}), nil
	}
	recognized, err := vault.Switch.GetVaults()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	list := e.source.GetPluginVaultList()
	out := make([]*v1.Vault, 0, len(recognized))
	for _, one := range recognized {
		out = append(out, &v1.Vault{
			Id: string(one.ID), Name: one.Name, Path: one.Path, IsMissing: list != nil && list.IsMissing(one), ManagedBy: one.ManagedBy,
		})
	}
	return connect.NewResponse(&v1.ListVaultsResponse{Vaults: out}), nil
}

func (e *vaultsCalls) AddVault(
	ctx context.Context, req *connect.Request[v1.AddVaultRequest],
) (*connect.Response[v1.AddVaultResponse], error) {
	pluginID, list, err := e.beginListAs(req.Header())
	if err != nil {
		return nil, err
	}
	managedBy := req.Msg.GetManagedBy()
	if managedBy != "" && managedBy != pluginID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errManagerNotCaller)
	}
	vault, addErr := list.AddVault(ctx, req.Msg.GetPath(), req.Msg.GetName(), managedBy)
	code, detail, err := getVaultsResult(addErr)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return connect.NewResponse(&v1.AddVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.AddVaultResponse{Vault: writeVault(vault)}), nil
}

func (e *vaultsCalls) RenameVault(
	ctx context.Context, req *connect.Request[v1.RenameVaultRequest],
) (*connect.Response[v1.RenameVaultResponse], error) {
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
		return connect.NewResponse(&v1.RenameVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.RenameVaultResponse{Vault: writeVault(vault)}), nil
}

func (e *vaultsCalls) RemoveVault(
	ctx context.Context, req *connect.Request[v1.RemoveVaultRequest],
) (*connect.Response[v1.RemoveVaultResponse], error) {
	pluginID, list, err := e.beginListAs(req.Header())
	if err != nil {
		return nil, err
	}
	if req.Msg.GetIsFolderTrashed() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoTrash)
	}
	id := domain.VaultID(req.Msg.GetId())
	// The folder is read from the list before the vault leaves it.
	entry, err := e.findVault(id)
	if err == nil && entry.ManagedBy != pluginID {
		return nil, connect.NewError(connect.CodePermissionDenied, errNotManager)
	}
	if err == nil {
		err = list.RemoveVault(ctx, id)
	}
	code, detail, err := getVaultsResult(err)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		return connect.NewResponse(&v1.RemoveVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.RemoveVaultResponse{Path: entry.Path}), nil
}

// OpenVault returns once the window shows the vault or has rejected the open. A rejection for unsaved
// changes returns UNSAVED with the window's message in detail.
func (e *vaultsCalls) OpenVault(
	ctx context.Context, req *connect.Request[v1.OpenVaultRequest],
) (*connect.Response[v1.OpenVaultResponse], error) {
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
		return connect.NewResponse(&v1.OpenVaultResponse{Error: code, Detail: detail}), nil
	}
	return connect.NewResponse(&v1.OpenVaultResponse{}), nil
}
