// Package hostclient reaches the vault, the plugin's folder of its store, the text versions and the
// queue through the clients of the host.
package hostclient

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

// VaultReader is what the plugin reads of a vault: its files and its name.
type VaultReader interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Host is what the plugin works through the host.
type Host struct {
	Vaults func(domain.VaultID) VaultReader
	Store  func(domain.VaultID) transcribe.Folder
	Texts  func(domain.VaultID) host.TextVersions
	Queue  host.Queue
}

// Open returns the clients of the host at the endpoint.
func Open(endpoint hostcall.Endpoint) Host {
	client := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	texts := host.NewTextClient(endpoint)
	return Host{
		Vaults: func(id domain.VaultID) VaultReader { return client.With(id) },
		Store:  func(id domain.VaultID) transcribe.Folder { return store.With(id) },
		Texts:  func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		Queue:  host.NewQueueClient(endpoint),
	}
}

// Files reads the recordings of the vaults through the host.
type Files struct {
	vaults func(domain.VaultID) VaultReader
}

// NewFiles returns the files of the vaults.
func NewFiles(vaults func(domain.VaultID) VaultReader) Files {
	return Files{vaults: vaults}
}

// ReadFile returns the bytes of the file in the vault.
func (f Files) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	raw, _, err := f.vaults(vault).Read(ctx, path)
	return raw, err
}

// GetActiveVaultID returns the id of the vault the host shows.
func (f Files) GetActiveVaultID(ctx context.Context) (domain.VaultID, error) {
	shown, err := f.vaults("").GetVault(ctx)
	if err != nil {
		return "", err
	}
	return shown.ID, nil
}
