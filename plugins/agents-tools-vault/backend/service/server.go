package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// Mount registers the vault tools on s, working the vault through the host the environment names.
func Mount(s *server.Server) error {
	return host.MountWithVault(s, mount)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return host.NewHandlerWithVault(vars, mount)
}

func mount(s *server.Server, vault host.Vault) error {
	registry, err := New(vault).Registry()
	if err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}
