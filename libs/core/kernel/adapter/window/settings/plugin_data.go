package settings

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// PluginData is the folders plugins keep in a vault's store.
type PluginData interface {
	GetPluginData(ctx context.Context, v domain.Vault, pluginID string) (plugin.DataSize, error)
	DeletePluginData(ctx context.Context, v domain.Vault, pluginID string) error
}

// VaultProvider names the vault the window shows. A window showing none returns a vault with no ID.
type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

var errNoVault = errors.New("the window shows no vault")

// GetPluginData is the size of a plugin's folder in the shown vault. A window
// showing no vault answers an empty folder.
func (h *PluginFolders) GetPluginData(
	ctx context.Context, r *connect.Request[v1.GetPluginDataRequest],
) (*connect.Response[v1.GetPluginDataResponse], error) {
	if h.data == nil || h.vaults == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	v := h.vaults.GetCurrentVault()
	if v.ID == "" {
		return connect.NewResponse(&v1.GetPluginDataResponse{}), nil
	}
	size, err := h.data.GetPluginData(ctx, v, r.Msg.GetPluginId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.GetPluginDataResponse{
		FileCount: int32(size.FileCount), ByteCount: size.ByteCount,
	}), nil
}

// DeletePluginData deletes a plugin's folder in the shown vault and the
// records of its text versions.
func (h *PluginFolders) DeletePluginData(
	ctx context.Context, r *connect.Request[v1.DeletePluginDataRequest],
) (*connect.Response[v1.DeletePluginDataResponse], error) {
	if h.data == nil || h.vaults == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	v := h.vaults.GetCurrentVault()
	if v.ID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	if err := h.data.DeletePluginData(ctx, v, r.Msg.GetPluginId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeletePluginDataResponse{}), nil
}
