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
	names  ports.Names
}

// New returns the vaults served by the clients of the host, with the names of their notes put by names.
func New(client *host.Client, store *host.StoreClient, names ports.Names) *Vaults {
	return &Vaults{client: client, store: store, names: names}
}

// GetVaults lists every vault of the installation.
func (v *Vaults) GetVaults(ctx context.Context) ([]domain.Vault, error) {
	return v.client.GetVaults(ctx)
}

func (v *Vaults) GetFiles(id domain.VaultID) ports.Files {
	return v.names.GetFiles(id, v.client.With(id))
}

func (v *Vaults) GetNotes(id domain.VaultID) ports.Notes {
	return v.names.GetNotes(id, v.client.With(id), v.client.With(id))
}

// Store returns the store folder of the plugin in the vault.
func (v *Vaults) Store(id domain.VaultID) ports.Store { return v.store.With(id) }

// GetAbsent returns the files of the vault that the device lists and does not hold.
func (v *Vaults) GetAbsent(id domain.VaultID) ports.Absent {
	return v.names.GetAbsent(id, v.client.With(id), v.client.With(id))
}

// GetTransfers returns what the host does with the bytes of the files of the vault.
func (v *Vaults) GetTransfers(id domain.VaultID) ports.Transfers {
	return v.names.GetTransfers(id, v.client.With(id), v.client.With(id))
}

// AddManagedVault adds the folder as a vault that only the plugin with the id managedBy writes.
func (v *Vaults) AddManagedVault(ctx context.Context, path, name, managedBy string) (domain.Vault, error) {
	return v.client.AddManagedVault(ctx, path, name, managedBy)
}

// RemoveVault takes the vault off the list of the installation and returns its folder, which stays on the disk.
func (v *Vaults) RemoveVault(ctx context.Context, id domain.VaultID) (string, error) {
	return v.client.RemoveVault(ctx, id)
}
