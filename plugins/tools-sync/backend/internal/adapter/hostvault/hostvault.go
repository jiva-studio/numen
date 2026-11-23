// Package hostvault is the vaults of the installation, reached through the host.
package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Vaults are the vaults of the installation, their files and their store folders.
type Vaults struct {
	client *host.Client
	store  *host.StoreClient
}

// New returns the vaults served by the clients of the host.
func New(client *host.Client, store *host.StoreClient) *Vaults {
	return &Vaults{client: client, store: store}
}

// GetVaults lists every vault of the installation.
func (v *Vaults) GetVaults(ctx context.Context) ([]domain.Vault, error) {
	return v.client.GetVaults(ctx)
}

// Files returns the files of the vault.
func (v *Vaults) Files(id domain.VaultID) ports.Files { return v.client.With(id) }

// Indexer returns the index of the vault.
func (v *Vaults) Indexer(id domain.VaultID) ports.Indexer { return v.client.With(id) }

// Store returns the store folder of the plugin in the vault.
func (v *Vaults) Store(id domain.VaultID) ports.Store { return v.store.With(id) }

// Absent returns the files of the vault that the device lists and does not hold.
func (v *Vaults) Absent(id domain.VaultID) ports.Absent { return v.client.With(id) }
