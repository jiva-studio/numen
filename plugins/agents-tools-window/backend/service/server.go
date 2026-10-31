package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// Mount registers the window tools on s, working the vault through the host the
// environment names.
func Mount(s *server.Server) error {
	return mount(s, connectVault(host.NewClientFromEnv()))
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, connectVault(host.NewClientFromVars(vars)))
	})
}

// connectVault is the vault of the host client, or no vault where the host cannot be reached.
func connectVault(client *host.Client, err error) host.Vault {
	if err != nil {
		return nil
	}
	return client
}

func mount(s *server.Server, vault host.Vault) error {
	manager := NewManager(clock.System{})
	registry := tools.New()
	if err := RegisterTools(registry, manager, vault); err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}
