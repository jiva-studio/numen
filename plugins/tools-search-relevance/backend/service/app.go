// Package service is the backend of the search relevance plugin: the service the settings view
// calls to measure and set the search floor, and the handler of the events that make the plugin
// measure a vault once its vectors exist.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/internal/adapter/hostclient"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/calibration"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.search.relevance"

// Hosts is what the plugin works through the host.
type Hosts = calibration.Hosts

// App is the backend of the plugin. NewApp and NewAppOver make one.
type App struct {
	open  func() (Hosts, error)
	clock relevance.Clock

	calibration *calibration.Calibration
}

// NewApp returns the backend that connects to the host the process was launched for when it is mounted.
func NewApp() *App {
	return &App{open: hostclient.OpenFromEnv, clock: clock.System{}, calibration: calibration.New()}
}

// NewAppOver returns the backend over the hosts given, with measurements stamped by the clock.
func NewAppOver(hosts Hosts, clk relevance.Clock) *App {
	return &App{
		open:  func() (Hosts, error) { return hosts, nil },
		clock: clk, calibration: calibration.New(),
	}
}

// Mount registers the service on the server.
func (a *App) Mount(s *server.Server) error {
	hosts, err := a.open()
	if err != nil {
		return err
	}
	if err := a.calibration.StartOver(hosts, a.clock); err != nil {
		return err
	}

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
