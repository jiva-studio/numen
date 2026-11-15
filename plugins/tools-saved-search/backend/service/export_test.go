package service

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
)

// NewAppOver returns the backend over the hosts given, with the settings of current and the time of clk.
func NewAppOver(hosts Hosts, current *settings.Store, clk ports.Clock) *App {
	app := &App{open: func() (Hosts, error) { return hosts, nil }, settings: current, clock: clk, parts: nil}
	app.parts = app.newParts(hosts)
	return app
}

// NewRegistry returns the registry of the tools of the plugin.
func (a *App) NewRegistry() (*tools.Registry, error) {
	built, err := a.getParts()
	if err != nil {
		return nil, err
	}
	return a.newRegistry(built)
}

func (a *App) Wait() {
	a.parts.runner.Wait()
}
