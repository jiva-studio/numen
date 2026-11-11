// Package service is the backend of the search relevance plugin: the service the settings view
// calls to measure and set the search floor, and the handler of the events that make the plugin
// measure a vault once its vectors exist.
package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/calibration"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/relevance"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.search.relevance"

var errIncompleteHost = errors.New("search relevance needs the search index, the store and the queue of the host")

// errNotMounted is a call that reached the app before it was mounted.
var errNotMounted = server.NewAPIError(server.CodeUnavailable, "the plugin is not ready")

// Hosts is what the plugin works through the host.
type Hosts struct {
	Floors relevance.Host
	Queue  host.Queue
}

// App is the backend of the plugin. NewApp and NewAppOver make one.
type App struct {
	open  func() (Hosts, error)
	clock relevance.Clock

	calibration *calibration.Calibration
}

// NewApp returns the backend that connects to the host the process was launched for when it is mounted.
func NewApp() *App {
	return &App{open: openFromEnv, clock: clock.System{}, calibration: calibration.New()}
}

// NewAppOver returns the backend over the hosts given, with measurements stamped by the clock.
func NewAppOver(hosts Hosts, clk relevance.Clock) *App {
	return &App{
		open:  func() (Hosts, error) { return hosts, nil },
		clock: clk, calibration: calibration.New(),
	}
}

func openFromEnv() (Hosts, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return Hosts{}, err
	}
	client := host.NewClient(endpoint)
	search := host.NewSearchClient(endpoint)
	store := host.NewStoreClient(endpoint)
	notes := func(id domain.VaultID) host.NoteLookup { return client.With(id) }
	return Hosts{
		Floors: relevance.Host{
			Search: func(id domain.VaultID) relevance.Search {
				on := search.With(id)
				return relevance.Search{Index: on, Policy: on}
			},
			Vaults: notes,
			Store:  func(id domain.VaultID) relevance.Folder { return store.With(id) },
		},
		Queue: host.NewQueueClient(endpoint),
	}, nil
}

// Mount registers the service on the server.
func (a *App) Mount(s *server.Server) error {
	hosts, err := a.open()
	if err != nil {
		return err
	}
	if hosts.Floors.Search == nil || hosts.Floors.Vaults == nil || hosts.Floors.Store == nil || hosts.Queue == nil {
		return errIncompleteHost
	}
	a.calibration.Start(relevance.New(hosts.Floors, a.clock), hosts.Queue)

	service := s.NewAPI(api.Service)
	server.Handle(service, api.MethodGetFloor, a.getFloor)
	server.Handle(service, api.MethodCalibrateFloor, a.calibrateFloor)
	server.Handle(service, api.MethodSetFloor, a.setFloor)
	server.Handle(service, api.MethodProbeFloor, a.probeFloor)
	// gomutants:disable-next-line RETURN_ERROR_NIL reason="Constant service and method names are always valid."
	return service.Err()
}

// HandleEvent hands the event of the host to the calibration.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	return a.calibration.HandleEvent(ctx, event)
}
