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

// VaultReader is what the plugin reads of a vault: its files, its file change feed and its name.
type VaultReader interface {
	host.FileReader
	host.FileChangeLister
	GetVault(ctx context.Context) (domain.Vault, error)
}

// Host is what the plugin works through the host.
type Host struct {
	GetVault        func(domain.VaultID) VaultReader
	Store           func(domain.VaultID) transcribe.Folder
	GetTexts        func(domain.VaultID) host.TextVersions
	Queue           host.Queue
	FetchDependency func(ctx context.Context, id string) (string, error)
}

// Open returns the clients of the host at the endpoint.
func Open(endpoint hostcall.Endpoint) Host {
	client := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	texts := host.NewTextClient(endpoint)
	return Host{
		GetVault:        func(id domain.VaultID) VaultReader { return client.With(id) },
		Store:           func(id domain.VaultID) transcribe.Folder { return store.With(id) },
		GetTexts:        func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		Queue:           host.NewQueueClient(endpoint),
		FetchDependency: host.NewDependencyClient(endpoint).FetchDependency,
	}
}

// Files reads the recordings of the vaults through the host.
type Files struct {
	getVault func(domain.VaultID) VaultReader
}

func NewFiles(vaults func(domain.VaultID) VaultReader) Files {
	return Files{getVault: vaults}
}

func (f Files) ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error) {
	raw, _, err := f.getVault(vault).Read(ctx, path)
	return raw, err
}

func (f Files) GetActiveVaultID(ctx context.Context) (domain.VaultID, error) {
	shown, err := f.getVault("").GetVault(ctx)
	return shown.ID, err
}
