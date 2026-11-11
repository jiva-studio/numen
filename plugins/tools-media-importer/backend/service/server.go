// Package service is the backend of the media importer plugin: it makes link notes, fetches the text
// their addresses publish as text versions of them, copies the videos they point at to be played
// in their place, and serves the importer API and the plugin's commands.
package service

import (
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/fetchers"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/tempfiles"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.media.importer"

func openHostAt(endpoint hostcall.Endpoint) link.Host {
	client := host.NewClient(endpoint)
	store := host.NewStoreClient(endpoint)
	texts := host.NewTextClient(endpoint)
	return link.Host{
		PluginID: PluginID,
		Vaults: func(id domain.VaultID) link.Vault {
			on := client.With(id)
			return link.Vault{Files: on, Writer: on}
		},
		Store: func(id domain.VaultID) link.Folder {
			on := store.With(id)
			return link.Folder{Files: on, Writer: on}
		},
		Texts:  func(id domain.VaultID) link.Texts { return texts.With(id) },
		Queue:  host.NewQueueClient(endpoint),
		Clock:  clock.System{},
		Report: process.Runner{}.ReportError,
	}
}

// Mount registers the importer API, the commands and the agent tool url_import on s over the host
// given. A video site is fetched through the mock downloader when isMock is set. The fetches end
// with s.
func Mount(s *server.Server, hosted link.Host, isMock bool) error {
	f := fetchers.New(isMock, process.Runner{}, tempfiles.Files{})
	links := link.New(hosted, f.GetFetcher, f.GetSettings)
	commands, err := newCommands(links)
	// gomutants:disable-next-line BRANCH_IF reason="newCommands with static command IDs never returns an error"
	if err != nil {
		return err
	}
	s.SetUpdateConfig(func(values map[string]string, _ server.Secrets) { f.UpdateConfig(values) })
	// gomutants:disable-next-line BRANCH_IF reason="mountAPI with static service and method constants never returns an error"
	if err := mountAPI(s, links); err != nil {
		return err
	}
	registry, err := NewTools(links).Registry()
	if err != nil {
		return err
	}
	command.Mount(s, commands)
	registry.Mount(s)
	s.OnClose(func() error {
		links.Close()
		return nil
	})
	return nil
}

// MountFromEnv mounts the plugin over the host the process was started for.
func MountFromEnv(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	return Mount(s, openHostAt(endpoint), environment.IsMockModel())
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	hosted := openHostAt(endpoint)
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return Mount(s, hosted, environment.IsMockModelIn(vars))
	})
}
