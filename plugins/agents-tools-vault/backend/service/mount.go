package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Mount registers the vault tools on s, working the vault through the host the environment names.
func Mount(s *server.Server) error {
	client, err := host.NewClientFromEnv()
	if err != nil {
		return err
	}
	texts, err := host.NewTextClientFromEnv()
	if err != nil {
		return err
	}
	search, err := host.NewSearchClientFromEnv()
	if err != nil {
		return err
	}
	return mount(s, client, texts, search)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		client, err := host.NewClientFromVars(vars)
		if err != nil {
			return err
		}
		texts, err := host.NewTextClientFromVars(vars)
		if err != nil {
			return err
		}
		search, err := host.NewSearchClientFromVars(vars)
		if err != nil {
			return err
		}
		return mount(s, client, texts, search)
	})
}

func mount(s *server.Server, client *host.Client, texts *host.TextClient, search *host.SearchClient) error {
	registry, err := New(createVaultGetter(client)).WithLibrary(createLibraryGetter(texts, search)).WithVaultList(client).Registry()
	if err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}

func createVaultGetter(client *host.Client) func(domain.VaultID) Vault {
	return func(id domain.VaultID) Vault {
		on := client.With(id)
		return Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Arranger: on, Editor: on, Selector: on}
	}
}

func createLibraryGetter(texts *host.TextClient, search *host.SearchClient) func(domain.VaultID) Library {
	return func(id domain.VaultID) Library {
		on := texts.With(id)
		return Library{Versions: on, Texts: on, Search: search.With(id)}
	}
}
