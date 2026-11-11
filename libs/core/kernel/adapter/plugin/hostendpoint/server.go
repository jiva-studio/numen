package hostendpoint

import (
	"context"
	"sync"
)

// Server opens the Endpoint on the first call to Open that finds something to serve, and closes it.
type Server struct {
	address string
	callers Callers

	mu       sync.Mutex
	endpoint serving
}

// NewServer creates a server that serves callers on address once Open finds a service set.
func NewServer(address string, callers Callers) *Server {
	return &Server{address: address, callers: callers}
}

// Open starts serving when the host lends the completer, the vault, the search, the queue of work or
// the dependencies, and returns the port. It returns port 0 while the host lends none of them, and
// the port it already serves on when it is open.
func (s *Server) Open(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.endpoint != nil {
		return s.endpoint.GetPort(), nil
	}
	if !s.hasServices() {
		return 0, nil
	}
	endpoint, err := Open(ctx, s.address, s.callers)
	if err != nil {
		return 0, err
	}
	s.endpoint = endpoint
	return endpoint.GetPort(), nil
}

func (s *Server) hasServices() bool {
	return s.callers.Lent.GetCompleter() != nil || s.callers.Lent.GetPluginVault() != nil ||
		s.callers.Dependencies() != nil || s.callers.Lent.GetPluginWork() != nil ||
		s.callers.Lent.GetPluginSearch() != nil
}

// GetPort returns the port the server serves on, or 0 while it is not open.
func (s *Server) GetPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.endpoint == nil {
		return 0
	}
	return s.endpoint.GetPort()
}

// Close stops serving and waits for the calls being served to end, up to ctx.
func (s *Server) Close(ctx context.Context) error {
	s.mu.Lock()
	endpoint := s.endpoint
	s.endpoint = nil
	s.mu.Unlock()
	if endpoint == nil {
		return nil
	}
	return endpoint.Close(ctx)
}

// ClearPlugin forgets the tasks and the turns of a plugin whose process has ended.
func (s *Server) ClearPlugin(pluginID string) {
	s.mu.Lock()
	endpoint := s.endpoint
	s.mu.Unlock()
	if endpoint != nil {
		endpoint.ClearPluginTasks(pluginID)
	}
	if work := s.callers.Lent.GetPluginWork(); work != nil {
		work.ClearPlugin(pluginID)
	}
}

// serving is an endpoint that answers the calls of plugins.
type serving interface {
	GetPort() int
	Close(ctx context.Context) error
	ClearPluginTasks(pluginID string)
}
