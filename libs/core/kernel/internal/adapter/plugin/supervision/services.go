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

// Services holds what the host gives the plugins it launches: the endpoint they call the host
// through and the provisioner of the dependencies they declare.
type Services struct {
	closing Closing

	mu           sync.RWMutex
	dependencies pluginport.DependencyRegistry
	endpoint     HostEndpoint
}

// NewServices returns services with nothing set; they open no endpoint once closing reports the host shut down.
func NewServices(closing Closing) *Services {
	return &Services{closing: closing}
}

// SetHostEndpoint sets the endpoint the plugins call the host through. The host opens it when the
// first plugin launches after it is set.
func (s *Services) SetHostEndpoint(endpoint HostEndpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.endpoint = endpoint
}

// SetDependencies sets what provisions the dependencies the plugins declare and
// what the plugins ask about over the host endpoint.
func (s *Services) SetDependencies(catalog pluginport.DependencyRegistry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dependencies = catalog
}

// GetDependencies returns what provisions the dependencies, or nil when nothing does.
func (s *Services) GetDependencies() pluginport.DependencyRegistry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dependencies
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
	s.mu.RLock()
	endpoint := s.endpoint
	s.mu.RUnlock()
	if endpoint == nil {
		return 0, nil
	}
	if port := endpoint.GetPort(); port != 0 || isClosed {
		return port, nil
	}
	return endpoint.Open(ctx)
}

func (s *Services) CloseHostEndpoint(ctx context.Context) error {
	s.mu.RLock()
	endpoint := s.endpoint
	s.mu.RUnlock()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

// ClearPlugin forgets what the endpoint holds for a plugin whose process ended.
func (s *Services) ClearPlugin(pluginID string) {
	s.mu.RLock()
	endpoint := s.endpoint
	s.mu.RUnlock()
	if endpoint != nil {
		endpoint.ClearPlugin(pluginID)
	}
}
