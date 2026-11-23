package supervision

import (
	"context"
	"maps"
	"slices"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// HostEndpoint is the endpoint the plugins call the host through. The host opens it when the first
// plugin launches, and passes the port to every plugin it launches.
type HostEndpoint interface {
	// GetPort returns the port the endpoint serves on, or 0 while it is not open.
	GetPort() int
	// Open starts the endpoint and returns its port. It returns 0 where the host has nothing to lend.
	Open(ctx context.Context) (int, error)
	Close(ctx context.Context) error
	// ClearPlugin forgets what the endpoint holds for a plugin whose process has ended.
	ClearPlugin(pluginID string)
}

// Closing reports whether the host is shut down.
type Closing interface {
	IsClosed() bool
}

// Slot holds one value that is replaced as a whole while readers use it.
type Slot[T any] struct {
	mu    sync.RWMutex
	value T
}

// Get returns the value held, the zero value until one is replaced in.
func (s *Slot[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Replace holds value from now on.
func (s *Slot[T]) Replace(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
}

// Dependencies is what the host reads of the provisioner of the dependencies the plugins declare.
type Dependencies interface {
	pluginport.DependencyRegistry
	CanProvision(req manifest.DependencyRequirement) bool
	// Resolve returns where the dependency is or will be and whether it is ready, without the network, a lock wait or a download.
	Resolve(req manifest.DependencyRequirement) (plugindomain.ResolvedDependency, plugindomain.DependencyState, error)
	// Cancel stops the download of the requirement and reports whether one was in progress.
	Cancel(req manifest.DependencyRequirement) bool
}

// Services holds what the host gives the plugins it launches: the endpoint they call the host
// through and the provisioner of the dependencies they declare. Both are read from the functions
// the services are made with, at each use.
type Services struct {
	closing      Closing
	getEndpoint     func() HostEndpoint
	getDependencies func() Dependencies

	mu          sync.Mutex
	observe    func(manifest.DependencyRequirement, plugindomain.DependencyStatus)
	downloading map[string]manifest.DependencyRequirement
}

// NewServices returns services that read the endpoint and the provisioner of the dependencies from the
// given functions; a nil result means there is none. They open no endpoint once closing reports the host shut down.
func NewServices(
	closing Closing, getEndpoint func() HostEndpoint, getDependencies func() Dependencies,
) *Services {
	return &Services{closing: closing, getEndpoint: getEndpoint, getDependencies: getDependencies, downloading: map[string]manifest.DependencyRequirement{}}
}

// GetDependencies returns what provisions the dependencies, or nil when nothing does.
func (s *Services) GetDependencies() Dependencies {
	return s.getDependencies()
}

// ObserveDependencies sets the function that observes each change of the state of a dependency. A
// host that provisions none observes none.
func (s *Services) ObserveDependencies(observe func(manifest.DependencyRequirement, plugindomain.DependencyStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observe = observe
}

// Observe records the change of the state of a dependency and passes it to the observer. A provisioner of
// dependencies takes it at its construction.
func (s *Services) Observe(req manifest.DependencyRequirement, status plugindomain.DependencyStatus) {
	s.mu.Lock()
	if status.State == plugindomain.DependencyStateProvisioning {
		s.downloading[req.GetIdentity()] = req
	} else {
		delete(s.downloading, req.GetIdentity())
	}
	observe := s.observe
	s.mu.Unlock()
	if observe != nil {
		observe(req, status)
	}
}

// CancelDependency stops the download of the dependency, if one runs.
func (s *Services) CancelDependency(req manifest.DependencyRequirement) {
	if catalog := s.GetDependencies(); catalog != nil {
		catalog.Cancel(req)
	}
}

// CancelDependencies stops every download that the observer saw start and has not seen end.
func (s *Services) CancelDependencies() {
	s.mu.Lock()
	running := slices.Collect(maps.Values(s.downloading))
	s.mu.Unlock()
	for _, req := range running {
		s.CancelDependency(req)
	}
}

// OpenHostEndpoint opens the endpoint unless the host is shut down, and returns its port, or 0 where
// there is no endpoint or nothing to lend through it.
func (s *Services) OpenHostEndpoint(ctx context.Context) (int, error) {
	isClosed := s.closing.IsClosed()
	endpoint := s.getEndpoint()
	if endpoint == nil {
		return 0, nil
	}
	if port := endpoint.GetPort(); port != 0 || isClosed {
		return port, nil
	}
	return endpoint.Open(ctx)
}

func (s *Services) CloseHostEndpoint(ctx context.Context) error {
	endpoint := s.getEndpoint()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

// ClearPlugin forgets what the endpoint holds for a plugin whose process ended.
func (s *Services) ClearPlugin(pluginID string) {
	endpoint := s.getEndpoint()
	if endpoint != nil {
		endpoint.ClearPlugin(pluginID)
	}
}
