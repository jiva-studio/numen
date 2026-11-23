package server

import (
	"context"
	"maps"
	"sync"

	"github.com/jiva-studio/numen/libs/shared/protocol"
)

// ProtocolVersion is the protocol version the SDK uses.
const ProtocolVersion = protocol.PluginProtocolVersion

// LifecycleOption configures a LifecycleServer instance.
type LifecycleOption func(*LifecycleServer)

// WithLifecycleShutdown sets what the Shutdown RPC runs before it signals termination. The RPC
// signals termination even when fn fails, and answers the failure.
func WithLifecycleShutdown(fn func() error) LifecycleOption {
	return func(s *LifecycleServer) {
		s.handleShutdown = fn
	}
}

// LifecycleServer holds the lifecycle state of a plugin: its configuration and its shutdown signal.
// Server serves it to the host as the plugin lifecycle service.
type LifecycleServer struct {
	shutdownCh      chan struct{}
	shutdownOnce    sync.Once
	mu              sync.RWMutex
	config          map[string]string
	handleConfigure func(ctx context.Context, values map[string]string) error
	handleShutdown  func() error
}

// NewLifecycleServer creates a lifecycle service handler with a shutdown signal channel.
func NewLifecycleServer(shutdownCh chan struct{}, opts ...LifecycleOption) *LifecycleServer {
	ls := &LifecycleServer{
		shutdownCh: shutdownCh,
		config:     make(map[string]string),
	}
	for _, opt := range opts {
		opt(ls)
	}
	return ls
}

func (s *LifecycleServer) shutdown() error {
	var closeErr error
	if s.handleShutdown != nil {
		closeErr = s.handleShutdown()
	}
	s.TriggerShutdown()
	return closeErr
}

func (s *LifecycleServer) configureWith(ctx context.Context, values map[string]string) error {
	s.mu.Lock()
	if s.config == nil {
		s.config = make(map[string]string)
	}
	maps.Copy(s.config, values)
	callback := s.handleConfigure
	s.mu.Unlock()

	if callback != nil {
		return callback(ctx, values)
	}
	return nil
}

// TriggerShutdown initiates server termination.
func (s *LifecycleServer) TriggerShutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdownCh)
	})
}

// ShutdownTrigger returns the read-only shutdown notification channel.
func (s *LifecycleServer) ShutdownTrigger() <-chan struct{} {
	return s.shutdownCh
}

// SetConfigure registers the callback invoked during Configure RPC calls.
func (s *LifecycleServer) SetConfigure(fn func(ctx context.Context, values map[string]string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handleConfigure = fn
}

// SetUpdateConfig registers a simplified configuration updater callback.
func (s *LifecycleServer) SetUpdateConfig(fn func(values map[string]string)) {
	s.SetConfigure(func(_ context.Context, values map[string]string) error {
		fn(values)
		return nil
	})
}

func (s *LifecycleServer) GetConfig(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.config[key]
	return val, ok
}

// AllConfigs returns a copy of stored configuration values.
func (s *LifecycleServer) AllConfigs() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]string, len(s.config))
	maps.Copy(res, s.config)
	return res
}

// ApplyConfig stores the configuration values and runs the configure callback with the values.
func (s *LifecycleServer) ApplyConfig(ctx context.Context, values map[string]string) (bool, error) {
	if err := s.configureWith(ctx, values); err != nil {
		return false, err
	}
	return true, nil
}

// RequestShutdown runs the shutdown callback and signals termination.
func (s *LifecycleServer) RequestShutdown(context.Context) (bool, error) {
	if err := s.shutdown(); err != nil {
		return false, err
	}
	return true, nil
}
