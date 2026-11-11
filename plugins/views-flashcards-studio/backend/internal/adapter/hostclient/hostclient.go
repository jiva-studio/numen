// Package hostclient builds the calls the plugin makes on the host: the vault, by identity, and the
// plugin's folder of the store of each vault.
package hostclient

import (
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
)

// ReportError writes a non-fatal error to stderr.
func ReportError(what string, err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "views-flashcards-studio: %s: %v\n", what, err)
}

// Clients is what the use cases reach the host through.
type Clients struct {
	Vaults  func(domain.VaultID) hostports.Vault
	Folders func(domain.VaultID) hostports.Folder
}

// NewFromEnv returns the clients of the host the environment names.
func NewFromEnv() (Clients, error) {
	client, err := host.NewClientFromEnv()
	if err != nil {
		return Clients{Vaults: nil, Folders: nil}, err
	}
	store, err := host.NewStoreClientFromEnv()
	if err != nil {
		return Clients{Vaults: nil, Folders: nil}, err
	}
	return newClients(client, store), nil
}

// NewFromVars returns the clients of the host the variables of a launched process name.
func NewFromVars(vars map[string]string) (Clients, error) {
	client, err := host.NewClientFromVars(vars)
	if err != nil {
		return Clients{Vaults: nil, Folders: nil}, err
	}
	store, err := host.NewStoreClientFromVars(vars)
	if err != nil {
		return Clients{Vaults: nil, Folders: nil}, err
	}
	return newClients(client, store), nil
}

func newClients(client *host.Client, store *host.StoreClient) Clients {
	return Clients{
		Vaults: func(id domain.VaultID) hostports.Vault {
			on := client.With(id)
			return hostports.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Cache: on, Selector: on}
		},
		Folders: func(id domain.VaultID) hostports.Folder { return store.With(id) },
	}
}
