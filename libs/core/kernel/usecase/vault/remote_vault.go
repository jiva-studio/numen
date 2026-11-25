package vault

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// RemoteVaultProvider is the plugin that keeps vaults remotely.
type RemoteVaultProvider interface {
	// ListRemoteVaults lists the remote vaults the provider offers this device.
	ListRemoteVaults(ctx context.Context) ([]vaultdomain.RemoteVault, error)
	// ConnectRemoteVault connects the remote vault to the registered vault with the id and returns once
	// the connection is made.
	ConnectRemoteVault(ctx context.Context, remoteID string, vaultID domain.VaultID) error
}

// ListRemoteVaults lists the vaults the remote vault provider offers.
type ListRemoteVaults struct{ provider RemoteVaultProvider }

// NewListRemoteVaults creates the listing over the provider.
func NewListRemoteVaults(provider RemoteVaultProvider) *ListRemoteVaults {
	return &ListRemoteVaults{provider: provider}
}

// Execute returns the remote vaults. No provider, a provider that is off and a provider that fails give an
// empty list: a remote vault is an offer, and none to offer is not a failure.
func (u *ListRemoteVaults) Execute(ctx context.Context) []vaultdomain.RemoteVault {
	listed, err := u.provider.ListRemoteVaults(ctx)
	if err != nil || listed == nil {
		return []vaultdomain.RemoteVault{}
	}
	return listed
}
