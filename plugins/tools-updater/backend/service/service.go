// Package service is the backend of the updater plugin: it asks the endpoint that publishes the
// release manifest for the newest build, compares it with the running one and serves the result to
// the plugin's frontend.
package service

import (
	"context"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/feed"
	"github.com/jiva-studio/numen/plugins/tools-updater/backend/release"
)

// Endpoints of the release feeds of the two channels.
const (
	StableEndpoint = "https://dl.numen.md/latest/latest.json"
	BetaEndpoint   = "https://dl.numen.md/beta/latest.json"
)

// ChannelBeta is the value of the channel setting that selects the beta feed.
const ChannelBeta = "beta"

// Fetcher requests the release manifest from an endpoint.
type Fetcher interface {
	Fetch(ctx context.Context, endpoint string) (release.Manifest, error)
}

// updater holds the endpoint the host configured and the result of the last check.
type updater struct {
	fetcher Fetcher

	mu           sync.Mutex
	isConfigured bool
	endpoint     string
	status       api.Status
}

func newUpdater(fetcher Fetcher) *updater {
	return &updater{
		fetcher: fetcher, mu: sync.Mutex{}, isConfigured: false, endpoint: "",
		status: api.Status{State: api.StateUnchecked, Update: nil},
	}
}

// getEndpoint returns the endpoint the settings name: the endpoint setting where it is set, and the
// feed of the channel otherwise.
func getEndpoint(values map[string]string) string {
	if endpoint := strings.TrimSpace(values["endpoint"]); endpoint != "" {
		return endpoint
	}
	if strings.TrimSpace(values["channel"]) == ChannelBeta {
		return BetaEndpoint
	}
	return StableEndpoint
}

// configure applies the settings the host supplies. A check runs only after the first call, and a
// change of the endpoint discards the result of the earlier checks.
func (u *updater) configure(values map[string]string) {
	endpoint := getEndpoint(values)
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.isConfigured && u.endpoint == endpoint {
		return
	}
	u.isConfigured = true
	u.endpoint = endpoint
	u.status = api.Status{State: api.StateUnchecked, Update: nil}
}

// getConfigured returns the endpoint, and false before the host has configured the plugin.
func (u *updater) getConfigured() (string, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.endpoint, u.isConfigured
}

func (u *updater) setStatus(endpoint string, status api.Status) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.endpoint == endpoint {
		u.status = status
	}
}

func (u *updater) getStatus(context.Context, api.GetStatusRequest) (api.Status, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.status, nil
}

// check requests the manifest and compares it with the running build. An endpoint that does not
// answer with a manifest is the unreachable state and no failure of the call.
func (u *updater) check(ctx context.Context, req api.CheckRequest) (api.Status, error) {
	endpoint, ok := u.getConfigured()
	if !ok {
		return api.Status{}, server.NewAPIError(server.CodeFailedPrecondition, "the plugin is not configured")
	}
	current, err := newCurrent(req.App)
	if err != nil {
		return api.Status{}, err
	}
	status := api.Status{State: api.StateUnreachable, Update: nil}
	manifest, fetchErr := u.fetcher.Fetch(ctx, endpoint)
	if fetchErr == nil {
		status = newStatus(manifest, current)
	}
	u.setStatus(endpoint, status)
	return status, nil
}

// newCurrent returns the running build the frontend reported.
func newCurrent(app api.App) (release.Current, error) {
	switch app.Platform {
	case release.PlatformDarwin, release.PlatformWindows, release.PlatformLinux:
	default:
		return release.Current{}, server.NewAPIError(server.CodeInvalidArgument, "unknown platform "+app.Platform)
	}
	if strings.TrimSpace(app.Version) == "" || app.Build < 0 {
		return release.Current{}, server.NewAPIError(server.CodeInvalidArgument, "the running version is unknown")
	}
	return release.Current{Version: app.Version, Build: release.Build(app.Build), Platform: app.Platform}, nil
}

// newStatus returns what the manifest says to the running build.
func newStatus(manifest release.Manifest, current release.Current) api.Status {
	update, ok := release.Evaluate(manifest, current)
	if !ok {
		return api.Status{State: api.StateCurrent, Update: nil}
	}
	return api.Status{State: api.StateAvailable, Update: newUpdate(update)}
}

func newUpdate(update release.Update) *api.Update {
	var file *api.File
	if update.File != nil {
		file = &api.File{
			Name: update.File.Name, Arch: update.File.Arch, Kind: update.File.Kind,
			URL: update.File.URL, Size: update.File.Size, SHA256: update.File.SHA256,
		}
	}
	return &api.Update{
		Version: update.Version, Build: int64(update.Build), Channel: update.Channel,
		ReleasedAt: update.ReleasedAt, NotesURL: update.NotesURL, DownloadURL: update.DownloadURL,
		File: file,
	}
}

// mount registers the updater API on s. The fetcher requests the release manifest.
func mount(s *server.Server, fetcher Fetcher) error {
	u := newUpdater(fetcher)
	s.SetUpdateConfig(u.configure)
	a := s.NewAPI(api.Service)
	server.Handle(a, api.MethodCheck, u.check)
	server.Handle(a, api.MethodGetStatus, u.getStatus)
	return a.Err()
}

// Mount registers the updater API on s. The manifest is requested over the network.
func Mount(s *server.Server) error {
	return mount(s, feed.New())
}
