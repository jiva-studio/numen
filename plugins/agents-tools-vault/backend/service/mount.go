package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

const listFunctionsTime = 3 * time.Second

// Mount registers the vault tools on s, working the vault through the host the environment names.
// warn receives the error of a query tool that starts without the functions of the search language.
func Mount(s *server.Server, warn func(error)) error {
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
	return mount(s, client, texts, search, warn)
}

func mount(s *server.Server, client *host.Client, texts *host.TextClient, search *host.SearchClient, warn func(error)) error {
	functions, err := listFunctions(search)
	if err != nil {
		warn(err)
	}
	registry, err := New(createVaultGetter(client)).WithLibrary(createLibraryGetter(texts, search), functions).
		WithFunctionLister(search.ListSearchFunctions).WithVaultList(client).BuildRegistry()
	if err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}

func listFunctions(search *host.SearchClient) (host.SearchFunctions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listFunctionsTime)
	defer cancel()
	functions, err := search.ListSearchFunctions(ctx)
	if err != nil {
		return host.SearchFunctions{}, fmt.Errorf("list the functions of the search language: %w", err)
	}
	return functions, nil
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
		searchOn := search.With(id)
		return Library{Versions: on, Texts: on, Search: searchOn, Queries: searchOn}
	}
}
