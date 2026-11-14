package supervision

import (
	"context"
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

// Services holds what the host gives the plugins it launches: the endpoint they call the host
// through and the provisioner of the dependencies they declare. Both are read from the functions
// the services are made with, at each use.
type Services struct {
	closing      Closing
	endpoint     func() HostEndpoint
	dependencies func() pluginport.DependencyRegistry
}

// NewServices returns services that read the endpoint and the provisioner of the dependencies from the
// given functions; a nil result means there is none. They open no endpoint once closing reports the host shut down.
func NewServices(
	closing Closing, endpoint func() HostEndpoint, dependencies func() pluginport.DependencyRegistry,
) *Services {
	return &Services{closing: closing, endpoint: endpoint, dependencies: dependencies}
}

// GetDependencies returns what provisions the dependencies, or nil when nothing does.
func (s *Services) GetDependencies() pluginport.DependencyRegistry {
	return s.dependencies()
}

// ObserveDependencies sets the function that observes each change of the state of a dependency. A
// host that provisions none observes none.
func (s *Services) ObserveDependencies(observer func(manifest.DependencyRequirement, plugindomain.DependencyStatus)) {
	if catalog := s.GetDependencies(); catalog != nil {
		catalog.SetObserver(observer)
	}
}

// OpenHostEndpoint opens the endpoint unless the host is shut down, and returns its port, or 0 where
// there is no endpoint or nothing to lend through it.
func (s *Services) OpenHostEndpoint(ctx context.Context) (int, error) {
	isClosed := s.closing.IsClosed()
	endpoint := s.endpoint()
	if endpoint == nil {
		return 0, nil
	}
	if port := endpoint.GetPort(); port != 0 || isClosed {
		return port, nil
	}
	return endpoint.Open(ctx)
}

func (s *Services) CloseHostEndpoint(ctx context.Context) error {
	endpoint := s.endpoint()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

// ClearPlugin forgets what the endpoint holds for a plugin whose process ended.
func (s *Services) ClearPlugin(pluginID string) {
	endpoint := s.endpoint()
	if endpoint != nil {
		endpoint.ClearPlugin(pluginID)
	}
}
