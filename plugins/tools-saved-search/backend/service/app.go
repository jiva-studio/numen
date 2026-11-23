// Package service is the backend of the saved search plugin: it builds the host clients, the settings and
// the runner of the saved searches, registers the tools and the service of the tab, and
// handles the index changes of the host.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/logging"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/refresh"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/savedsearch"
)

// Hosts is what the plugin works through the host: the vaults it opens, the window and the task list.
type Hosts struct {
	Open   func(id domain.VaultID) refresh.Vault
	Window ports.Window
	Tasks  ports.Tasks
}

// App is the backend of the plugin. NewApp makes one.
type App struct {
	open     func() (Hosts, error)
	settings *settings.Store
	clock    ports.Clock
	parts    *parts
}

// parts are what the hosts are worked through, built once.
type parts struct {
	hosts  Hosts
	runner *refresh.Runner
	notes  *savedsearch.UseCase
}

// NewApp returns the backend that connects to the host the process was launched for when it is mounted;
// the server mounts it before it serves a call or an event.
func NewApp() *App {
	return &App{open: openFromEnv, settings: settings.NewStore(), clock: clock.System{}, parts: nil}
}

// Mount registers the tools, the settings and the service of the tab on the server.
func (a *App) Mount(s *server.Server) error {
	built, err := a.getParts()
	if err != nil {
		return err
	}
	registry, err := a.newRegistry(built)
	if err != nil {
		return err
	}
	s.SetConfigure(func(ctx context.Context, values map[string]string) error {
		a.updateSettings(ctx, built, values)
		return nil
	})
	commands, err := newCommands(built)
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	service := s.NewAPI(api.Service)
	server.Handle(service, api.MethodGetResults, func(ctx context.Context, req api.GetResultsRequest) (api.GetResultsResponse, error) {
		return getResults(ctx, built, req)
	})
	server.Stream(service, api.MethodWatchResults, func(ctx context.Context, req api.WatchResultsRequest, send func(api.ResultsChange) error) error {
		return watchResults(ctx, built, req, send)
	})
	server.Handle(service, api.MethodUpdateSearch, func(ctx context.Context, req api.UpdateSearchRequest) (api.UpdateSearchResponse, error) {
		return updateSearch(ctx, built, req)
	})
	server.Handle(service, api.MethodListSearchFunctions, func(ctx context.Context, _ api.ListSearchFunctionsRequest) (api.ListSearchFunctionsResponse, error) {
		return listSearchFunctions(ctx, built)
	})
	return service.Err()
}

// HandleEvent marks the saved searches of a vault whose index changed stale and starts the run of the ones
// that are due in the background. Every other event is ignored.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	vault, isIndexChange := host.ReadIndexChangedEvent(event)
	if !isIndexChange {
		return nil
	}
	built, err := a.getParts()
	if err != nil {
		return err
	}
	if err := built.runner.MarkStale(ctx, vault); err != nil {
		return err
	}
	built.runner.StartRunDue(context.WithoutCancel(ctx), vault)
	return nil
}

func (a *App) updateSettings(ctx context.Context, built *parts, values map[string]string) {
	a.settings.Update(values)
	for _, vault := range built.runner.ListVaults() {
		built.runner.StartRunDue(context.WithoutCancel(ctx), vault)
	}
}

func (a *App) getParts() (*parts, error) {
	if a.parts != nil {
		return a.parts, nil
	}
	hosts, err := a.open()
	if err != nil {
		return nil, err
	}
	a.parts = a.newParts(hosts)
	return a.parts, nil
}

func (a *App) newParts(hosts Hosts) *parts {
	runner := refresh.New(refresh.Dependencies{Open: hosts.Open, Tasks: hosts.Tasks, Log: logging.Log{}}, a.settings.GetSettings, a.clock)
	notes := savedsearch.New(savedsearch.Dependencies{
		Open:     func(id domain.VaultID) ports.Notes { return hosts.Open(id).Notes },
		Runner:   runner,
		GetSettings: a.getNoteSettings,
		Clock:    a.clock,
		Entropy:  random.Entropy{},
	})
	return &parts{hosts: hosts, runner: runner, notes: notes}
}

func (a *App) getNoteSettings() savedsearch.Settings {
	return savedsearch.Settings{DefaultFolder: a.settings.GetDefaultFolder()}
}

func openFromEnv() (Hosts, error) {
	client, err := host.NewClientFromEnv()
	if err != nil {
		return Hosts{}, err
	}
	search, err := host.NewSearchClientFromEnv()
	if err != nil {
		return Hosts{}, err
	}
	store, err := host.NewStoreClientFromEnv()
	if err != nil {
		return Hosts{}, err
	}
	queue, err := host.NewQueueClientFromEnv()
	if err != nil {
		return Hosts{}, err
	}
	open := func(id domain.VaultID) refresh.Vault {
		vault := client.With(id)
		return refresh.Vault{Notes: vault, Lookup: vault, Search: search.With(id), Store: store.With(id)}
	}
	return Hosts{Open: open, Window: client, Tasks: queue}, nil
}
