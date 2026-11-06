package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// Mount registers the window tools on s, using the vault and the window of the host the
// environment names.
func Mount(s *server.Server) error {
	return mount(s, connectHost(hostcall.FromEnv()))
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, connectHost(hostcall.FromVars(vars)))
	})
}

// connectHost returns the vault and the window of the host at the endpoint, or none when the
// endpoint is not configured.
func connectHost(endpoint hostcall.Endpoint, err error) Host {
	if err != nil {
		return Host{}
	}
	client := host.NewClient(endpoint)
	return Host{Vault: client, Window: client}
}

func mount(s *server.Server, on Host) error {
	manager := NewManager(clock.System{})
	registry := tools.New()
	if err := RegisterTools(registry, manager, on); err != nil {
		return err
	}
	registry.Mount(s)
	return nil
}
