package updates

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/plugins/tools-updater/backend/domain/release"
)

// ErrNotConfigured is a check before the host supplied the settings.
var ErrNotConfigured = errors.New("the plugin is not configured")

// ErrUnknownVersion is a running build that names no version or a negative build.
var ErrUnknownVersion = errors.New("the running version is unknown")

// PlatformError is a running build on a platform the plugin does not know.
type PlatformError struct {
	Platform string
}

func (e *PlatformError) Error() string { return "unknown platform " + e.Platform }

// State is what the last check found.
type State string

// The states of the last check.
const (
	StateUnchecked   State = "unchecked"
	StateCurrent     State = "current"
	StateAvailable   State = "available"
	StateUnreachable State = "unreachable"
)

// App is the running build.
type App struct {
	Version  string
	Build    int64
	Platform string
}

// Status is the result of the last check. Update is set in the available state.
type Status struct {
	State  State
	Update *release.Update
}

// Fetcher requests the release manifest from an endpoint.
type Fetcher interface {
	Fetch(ctx context.Context, endpoint string) (release.Manifest, error)
}

// Checker holds the endpoint the host configured and the result of the last check.
type Checker struct {
	fetcher Fetcher

	mu           sync.Mutex
	isConfigured bool
	endpoint     string
	status       Status
}

// NewChecker returns a Checker that requests manifests with fetcher.
func NewChecker(fetcher Fetcher) *Checker {
	return &Checker{
		fetcher: fetcher, mu: sync.Mutex{}, isConfigured: false, endpoint: "",
		status: Status{State: StateUnchecked, Update: nil},
	}
}

// Configure selects the endpoint the settings name. A new endpoint discards the last result.
func (c *Checker) Configure(values map[string]string) {
	endpoint := GetEndpoint(values)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isConfigured && c.endpoint == endpoint {
		return
	}
	c.isConfigured = true
	c.endpoint = endpoint
	c.status = Status{State: StateUnchecked, Update: nil}
}

// GetStatus returns the result of the last check without a request to the endpoint.
func (c *Checker) GetStatus() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Check requests the manifest of the configured endpoint and compares it with the running build. An
// endpoint that does not answer with a manifest, or an empty one, leaves the state unreachable.
func (c *Checker) Check(ctx context.Context, app App) (Status, error) {
	endpoint, ok := c.getConfigured()
	if !ok {
		return Status{}, ErrNotConfigured
	}
	current, err := newCurrent(app)
	if err != nil {
		return Status{}, err
	}
	status := Status{State: StateUnreachable, Update: nil}
	if endpoint != "" {
		manifest, fetchErr := c.fetcher.Fetch(ctx, endpoint)
		if fetchErr == nil {
			status = newStatus(manifest, current)
		}
	}
	c.setStatus(endpoint, status)
	return status, nil
}

func (c *Checker) getConfigured() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.endpoint, c.isConfigured
}

func (c *Checker) setStatus(endpoint string, status Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.endpoint == endpoint {
		c.status = status
	}
}

func newCurrent(app App) (release.Current, error) {
	switch app.Platform {
	case release.PlatformDarwin, release.PlatformWindows, release.PlatformLinux:
	default:
		return release.Current{}, &PlatformError{Platform: app.Platform}
	}
	if strings.TrimSpace(app.Version) == "" || app.Build < 0 {
		return release.Current{}, ErrUnknownVersion
	}
	return release.Current{Version: app.Version, Build: release.Build(app.Build), Platform: app.Platform}, nil
}

func newStatus(manifest release.Manifest, current release.Current) Status {
	update, ok := release.Evaluate(manifest, current)
	if !ok {
		return Status{State: StateCurrent, Update: nil}
	}
	return Status{State: StateAvailable, Update: &update}
}
