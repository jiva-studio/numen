package service

import (
	"context"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/account"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	accountadapter "github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/account"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hoststore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/hostvault"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
)

const (
	settingServer         = "server"
	settingInterval       = "syncIntervalSeconds"
	settingRemovalDelay   = "removalDelaySeconds"
	settingForeground     = "syncOnlyWhileOpen"
	settingReceivedFolder = "receivedFolder"
	settingEpochRoot      = "epochRootKey"
	requestTimeout        = 30 * time.Second
	streamHeaderTimeout   = 30 * time.Second
)

// App is the backend of Numen Sync: Mount registers its methods and keeps the sync running, and HandleEvent wakes
// the sync of a vault whose files changed.
type App struct {
	runtime *runtime
}

// NewApp returns the backend, not yet mounted.
func NewApp() *App { return &App{runtime: nil} }

// Mount registers the methods of the settings view, and keeps this device signed in to the server of the setting
// server and its vaults in sync from the moment the host configures the plugin. It starts again over the new
// address when the setting changes.
func (a *App) Mount(s *server.Server) error {
	endpoint, err := s.GetHostEndpoint()
	if err != nil {
		return err
	}
	vaults := hostvault.New(host.NewClient(endpoint), host.NewStoreClient(endpoint))
	client := account.NewClient(endpoint)
	a.runtime = newRuntime(parts{
		identity: accountadapter.New(client),
		client:   client,
		health:   health.NewClient(endpoint),
		clock:    clock.System{},
		vaults:   vaults,
		state:    hoststore.New(vaults),
	})
	service := s.NewAPI(api.Service)
	mountMethods(service, a.runtime)
	if err := service.Err(); err != nil {
		return err
	}
	s.SetUpdateConfig(a.runtime.configure)
	s.AddCloser(func() error {
		a.runtime.stop()
		return nil
	})
	return nil
}

// HandleEvent wakes the sync of the vault an onFilesChanged event names, and ignores any other event.
func (a *App) HandleEvent(_ context.Context, event string) error {
	if a.runtime == nil {
		return nil
	}
	if vault, isChange := host.ReadFilesChangedEvent(event); isChange {
		a.runtime.wake(vault)
	}
	return nil
}

func newHTTPClient() *http.Client { return &http.Client{Timeout: requestTimeout} }

func newStreamHTTPClient() *http.Client { return loops.NewStreamHTTP(streamHeaderTimeout) }
