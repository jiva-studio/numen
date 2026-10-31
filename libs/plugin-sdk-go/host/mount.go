package host

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// MountWithVault mounts the services of a plugin on s, working the vault through the host the
// environment names.
func MountWithVault(s *server.Server, mount func(*server.Server, Vault) error) error {
	client, err := NewClientFromEnv()
	if err != nil {
		return err
	}
	return mount(s, client)
}

// NewHandlerWithVault returns the handler of a plugin's backend for a host that links the plugin
// in, working the vault through the host the variables name.
func NewHandlerWithVault(vars map[string]string, mount func(*server.Server, Vault) error) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		client, err := NewClientFromVars(vars)
		if err != nil {
			return err
		}
		return mount(s, client)
	})
}
