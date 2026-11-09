package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
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
	return mount(s, client, client, texts, search)
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
		return mount(s, client, client, texts, search)
	})
}

func mount(s *server.Server, vault host.Vault, list host.VaultList, texts host.Texts, search host.Search) error {
	registry, err := New(vault).WithLibrary(texts, search).WithVaultList(list).Registry()
	if err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}
