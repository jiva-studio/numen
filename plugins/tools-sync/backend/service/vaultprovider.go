package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/vaultprovider"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/providerwire"
)

// vaultProvider serves the cloud vaults of the account to the core as the vault provider.
type vaultProvider struct{ runtime *runtime }

func newVaultProvider(r *runtime) vaultProvider { return vaultProvider{runtime: r} }

// ListRemoteVaults returns the cloud vaults of the account, or none while the plugin cannot reach them: no server, no identity, no sign-in, an unreachable server.
func (p vaultProvider) ListRemoteVaults(ctx context.Context) ([]vaultprovider.RemoteVault, error) {
	remote, err := handle(ctx, p.runtime, api.MethodListRemoteVaults, func(b *Backend) (api.ListRemoteVaultsResponse, error) {
		return b.ListRemoteVaults(ctx, api.ListRemoteVaultsRequest{})
	})
	if err != nil {
		if providerwire.IsOffline(err) {
			return []vaultprovider.RemoteVault{}, nil
		}
		return nil, providerwire.NewError(err)
	}
	local, err := handle(ctx, p.runtime, api.MethodListVaults, func(b *Backend) (api.ListVaultsResponse, error) {
		return b.ListVaults(ctx, api.ListVaultsRequest{})
	})
	if err != nil {
		return nil, providerwire.NewError(err)
	}
	return providerwire.JoinRemoteVaults(remote.Vaults, local.Vaults), nil
}

// ConnectRemoteVault connects the empty local vault to the cloud vault and starts the download, as the command to get a vault from the cloud does.
func (p vaultProvider) ConnectRemoteVault(ctx context.Context, remoteID string, vaultID domain.VaultID) error {
	_, err := handle(ctx, p.runtime, api.MethodConnectVault, func(b *Backend) (api.ConnectVaultResponse, error) {
		response, err := b.ConnectVault(ctx, api.ConnectVaultRequest{VaultID: string(vaultID), Mode: api.ModeExisting, RemoteID: remoteID})
		if err == nil {
			p.runtime.wake(vaultID)
		}
		return response, err
	})
	return providerwire.NewError(err)
}
