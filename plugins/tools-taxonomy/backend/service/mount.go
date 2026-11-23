// Package service is the backend of the tags plugin: it builds the host clients and the cache of the
// tags, registers the tool tag_list, the query functions hasTag, hasTagExactly and hasAnyTag and the service of its
// frontend, and brings the tags up to date on each onFilesChanged event.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/settags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

// Open returns the frontmatter, the feed and the store folder of a vault.
type Open func(id domain.VaultID) tagcache.Vault

func openHostAt(endpoint hostcall.Endpoint) Open {
	client := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	return func(id domain.VaultID) tagcache.Vault {
		vault := client.With(id)
		return tagcache.Vault{Frontmatter: vault, Feed: vault, Store: store.With(id), Properties: vault}
	}
}

// App is the tool and the query functions of the plugin and its handler of the events of the host, over one cache of tags.
type App struct {
	cache     *tagcache.Caches
	setter    *settags.UseCase
	tools     *tools.Registry
	functions *searchfunction.Registry
}

// NewApp returns the plugin over the vaults open returns.
func NewApp(open Open) (*App, error) {
	cache := tagcache.New(open)
	registry, err := newTools(cache)
	if err != nil {
		return nil, err
	}
	functions, err := newFunctions(cache)
	if err != nil {
		return nil, err
	}
	setter := settags.New(cache, func(id domain.VaultID) settags.Vault {
		vault := open(id)
		return settags.Vault{Properties: vault.Properties, Store: vault.Store}
	})
	return &App{cache: cache, setter: setter, tools: registry, functions: functions}, nil
}

// Run serves the plugin over the host the environment names.
func Run() {
	var app *App
	server.Run(func(s *server.Server) error {
		endpoint, err := hostcall.ReadEnv()
		if err != nil {
			return err
		}
		app, err = NewApp(openHostAt(endpoint))
		if err != nil {
			return err
		}
		app.Mount(s)
		return nil
	}, server.WithEventHandler(func(ctx context.Context, event string) error { return app.HandleEvent(ctx, event) }))
}

// Mount registers the tool, the query functions and the service of the frontend on s.
func (a *App) Mount(s *server.Server) {
	a.Tools().Mount(s)
	a.Functions().Mount(s)
	mountAPI(s, a.cache, a.setter)
}

// Tools is the registry of the tool tag_list.
func (a *App) Tools() *tools.Registry { return a.tools }

// HandleEvent brings the tags of the vault an onFilesChanged event names up to date with its file change
// feed, and ignores any other event.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	id, ok := host.ReadFilesChangedEvent(event)
	if !ok {
		return nil
	}
	return a.cache.UpdateVault(ctx, id)
}

// Functions is the registry of the query functions hasTag, hasTagExactly and hasAnyTag.
func (a *App) Functions() *searchfunction.Registry { return a.functions }
