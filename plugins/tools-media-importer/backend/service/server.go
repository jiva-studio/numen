// Package service is the backend of the media importer plugin: it makes link notes, fetches the text
// their addresses publish as text versions of them, copies the videos they point at to be played
// in their place, and serves the importer API and the plugin's commands.
package service

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/clock"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// EnvMockML names the variable that has the plugin reach video sites through the mock downloader.
const EnvMockML = "NUMEN_MOCK_ML"

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.media.importer"

// openHostAt connects to the host at the endpoint.
func openHostAt(endpoint hostcall.Endpoint) link.Host {
	return link.Host{
		Vault: host.NewClient(endpoint),
		Store: host.NewStoreClient(endpoint),
		Texts: host.NewTextClient(endpoint),
		Queue: host.NewQueueClient(endpoint),
		Now:   clock.Now,
	}
}

// fetchers holds the downloader and the settings, made again once a setting about them changes.
type fetchers struct {
	mu       sync.RWMutex
	isMock   bool
	cfg      download.Config
	current  *download.Downloader
	settings link.Settings
}

func newFetchers(isMock bool) *fetchers {
	f := &fetchers{
		mu: sync.RWMutex{}, isMock: isMock, cfg: download.Defaults(), current: nil,
		settings: link.Settings{Captions: nil, IsAutomatic: true, CopyMaxSize: 500 << 20},
	}
	f.current = f.open()
	return f
}

func (f *fetchers) open() *download.Downloader {
	if f.isMock {
		return download.NewMock()
	}
	return download.New(f.cfg)
}

func (f *fetchers) getFetcher() link.Fetcher {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.current
}

func (f *fetchers) getSettings() link.Settings {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.settings
}

// updateConfig applies the settings the host supplies. A size that is not a number is left unchanged.
func (f *fetchers) updateConfig(values map[string]string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := values["yt_dlp"]; ok {
		f.cfg.Video = getTool(v)
	}
	if v, ok := values["ffmpeg"]; ok {
		f.cfg.Sound = getTool(v)
	}
	if v, ok := values["captions"]; ok {
		f.settings.Captions = getLanguages(v)
	}
	if v, ok := values["automatic_captions"]; ok {
		f.settings.IsAutomatic = v != "false"
	}
	if mb, err := strconv.ParseInt(values["copy_max_size_mb"], 10, 64); err == nil && mb >= 0 {
		f.settings.CopyMaxSize = mb << 20
	}
	f.current = f.open()
}

func getTool(path string) download.Tool {
	if path == "" {
		return download.Tool{Command: nil, Arguments: nil, Environment: nil}
	}
	return download.Tool{Command: []string{path}, Arguments: nil, Environment: nil}
}

// getLanguages is the languages a setting names, separated by commas, best first.
func getLanguages(value string) []string {
	var out []string
	for _, one := range strings.Split(value, ",") {
		if one = strings.TrimSpace(one); one != "" {
			out = append(out, one)
		}
	}
	return out
}

// Mount registers the importer API and the commands on s over the host given. A
// video site is reached through the mock downloader where isMock is set. The fetches end with s.
func Mount(s *server.Server, hosted link.Host, isMock bool) error {
	f := newFetchers(isMock)
	links := link.New(hosted, f.getFetcher, f.getSettings)
	commands, err := newCommands(links, hosted)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(f.updateConfig)
	mountAPI(s, links)
	command.Mount(s, commands)
	s.RegisterRoute(embedPattern, http.HandlerFunc(serveEmbed))
	s.OnClose(links.Close)
	return nil
}

// MountFromEnv mounts the plugin over the host the process was started for.
func MountFromEnv(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	return Mount(s, openHostAt(endpoint), os.Getenv(EnvMockML) == trueValue)
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
		return Mount(s, hosted, vars[EnvMockML] == trueValue)
	})
}
