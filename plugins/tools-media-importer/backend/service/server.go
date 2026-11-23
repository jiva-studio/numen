// Package service is the backend of the media importer plugin: it makes link notes, fetches the text
// their addresses publish as text versions of them, copies the videos they point at to be played
// in their place, and serves the importer API and the plugin's commands.
package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/tempfiles"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/fetchers"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.media.importer"

func openHostAt(endpoint hostcall.Endpoint) link.Host {
	client := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	texts := host.NewTextClient(endpoint)
	return link.Host{
		PluginID: PluginID,
		GetVault: func(id domain.VaultID) link.Vault {
			on := client.With(id)
			return link.Vault{Files: on, Writer: on, Feed: on}
		},
		Store: func(id domain.VaultID) link.Folder {
			on := store.With(id)
			return link.Folder{Files: on, Writer: on}
		},
		GetTexts: func(id domain.VaultID) link.Texts { return texts.With(id) },
		Queue:    host.NewQueueClient(endpoint),
		Clock:    clock.System{},
		Entropy:  random.Entropy{},
		Report:   process.Runner{}.ReportError,
	}
}

// App is the backend of the plugin: Mount registers its services, and HandleEvent downloads the missing
// captions the file change feed names on each onFilesChanged event.
type App struct {
	links *link.Links
}

func NewApp() *App { return &App{links: nil} }

// HandleEvent downloads, in the background, the missing captions of the link notes the file change feed
// names on an onFilesChanged event, and ignores any other event.
func (a *App) HandleEvent(ctx context.Context, event string) error {
	if _, ok := host.ReadFilesChangedEvent(event); ok && a.links != nil {
		a.links.StartMissingTextFetch(ctx, false)
	}
	return nil
}

// Mount registers the importer API, the commands and the agent tool url_import on s over the host
// given. A video site is fetched through the mock downloader when isMock is set. The fetches end
// with s.
func (a *App) Mount(s *server.Server, hosted link.Host, isMock bool) error {
	tools := download.Tools{Runner: process.Runner{}, Files: tempfiles.Files{}, Paths: tempfiles.Files{}, Web: webpage.New()}
	f := fetchers.New(func(programs fetch.Config) ports.Fetcher { return download.Open(isMock, tools, programs) })
	links := link.New(hosted, f.GetFetcher, f.GetSettings)
	a.links = links
	commands, err := newCommands(links)
	// gomutants:disable-next-line BRANCH_IF reason="newCommands with static command IDs never returns an error"
	if err != nil {
		return err
	}
	s.SetConfigure(func(ctx context.Context, values map[string]string) error {
		links.Configure(ctx, func() { f.UpdateConfig(values) })
		return nil
	})
	// gomutants:disable-next-line BRANCH_IF reason="mountAPI with static service and method constants never returns an error"
	if err := mountAPI(s, links); err != nil {
		return err
	}
	registry, err := NewTools(links).BuildRegistry()
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	s.AddCloser(func() error {
		links.Close()
		return nil
	})
	return nil
}

// MountFromEnv mounts the plugin over the host the process was started for.
func (a *App) MountFromEnv(s *server.Server) error {
	endpoint, err := hostcall.ReadEnv()
	if err != nil {
		return err
	}
	return a.Mount(s, openHostAt(endpoint), environment.IsMockModel())
}
