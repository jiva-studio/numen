package service

import (
	"net/http"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
)

// Mount registers the window tools on s, working the vault through the host the
// environment names.
func Mount(s *server.Server) error {
	client, _ := host.NewClientFromEnv()
	return mount(s, client)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		client, _ := host.NewClientFromVars(vars)
		return mount(s, client)
	})
}

func mount(s *server.Server, vault host.Vault) error {
	manager := NewManager()
	registry := tools.New()
	if err := RegisterTools(registry, manager, vault); err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}
