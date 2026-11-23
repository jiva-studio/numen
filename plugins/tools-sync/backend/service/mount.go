package service

import (
	"context"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/keeper"
	serveradapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/session"
)

const (
	settingServer  = "server"
	requestTimeout = 30 * time.Second
)

// Mount keeps the device signed in to the server of the setting server from the moment the host
// configures the plugin, and starts again over the new address when the setting changes.
func Mount(s *server.Server) error {
	endpoint, err := s.GetHostEndpoint()
	if err != nil {
		return err
	}
	deps := session.Deps{
		Identity: accountadapter.New(account.NewClient(endpoint)),
		Health:   health.NewClient(endpoint),
		Server:   nil,
		Clock:    clock.System{},
	}
	root, cancelRoot := context.WithCancel(context.Background())
	signIn := keeper.New(deps, newServer)
	s.SetUpdateConfig(func(values map[string]string) { signIn.Configure(root, values[settingServer]) })
	s.AddCloser(func() error {
		cancelRoot()
		signIn.Stop()
		return nil
	})
	return nil
}

func newServer(address string, now func() time.Time) session.Server {
	return serveradapter.New(serveradapter.Config{Address: address, HTTP: &http.Client{Timeout: requestTimeout}, Now: now})
}
