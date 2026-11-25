package vaultbridge

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// CapabilityResolver finds the enabled plugin that provides a capability.
type CapabilityResolver interface {
	ResolveCapability(capability string) (manifest.Discovered, error)
}

// Provider calls the VaultProviderService of the enabled plugin with the capability vault.provider, which
// it activates for the call.
type Provider struct {
	activator bridge.Activator
	plugins   CapabilityResolver
}

// NewProvider returns the provider over the plugins the activator starts.
func NewProvider(activator bridge.Activator, plugins CapabilityResolver) *Provider {
	return &Provider{activator: activator, plugins: plugins}
}

// ListRemoteVaults asks the plugin for the remote vaults it offers this device.
func (p *Provider) ListRemoteVaults(ctx context.Context) ([]vaultdomain.RemoteVault, error) {
	var listed []vaultdomain.RemoteVault
	err := call(ctx, p, "list remote vaults", func(client numenv1connect.VaultProviderServiceClient) error {
		answer, err := client.ListRemoteVaults(ctx, connect.NewRequest(&v1.ListRemoteVaultsRequest{}))
		if err != nil {
			return err
		}
		for _, one := range answer.Msg.GetVaults() {
			listed = append(listed, vaultdomain.RemoteVault{
				RemoteID: one.GetRemoteId(), Name: one.GetName(), SizeBytes: one.GetSizeBytes(), VaultID: domain.VaultID(one.GetVaultId()),
			})
		}
		return nil
	})
	return listed, err
}

// ConnectRemoteVault asks the plugin to connect the remote vault to the vault of this device.
func (p *Provider) ConnectRemoteVault(ctx context.Context, remoteID string, vaultID domain.VaultID) error {
	return call(ctx, p, "connect remote vault", func(client numenv1connect.VaultProviderServiceClient) error {
		_, err := client.ConnectRemoteVault(ctx, connect.NewRequest(&v1.ConnectRemoteVaultRequest{RemoteId: remoteID, VaultId: string(vaultID)}))
		return err
	})
}

func call(ctx context.Context, p *Provider, why string, run func(numenv1connect.VaultProviderServiceClient) error) (err error) {
	found, err := p.plugins.ResolveCapability(bridge.CapabilityVaultProvider)
	if err != nil {
		return err
	}
	client, lease, err := bridge.Open(ctx, p.activator, found.Manifest.ID, why, numenv1connect.NewVaultProviderServiceClient)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	return run(client)
}
