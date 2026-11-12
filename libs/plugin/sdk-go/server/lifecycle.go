package server

import (
	"context"
	"maps"
	"sync"
)

// ProtocolVersion is the protocol version the SDK uses.
const ProtocolVersion = 1

// statusOK is the status a healthy plugin answers a ping with.
const statusOK = "ok"

// LifecycleOption configures a LifecycleServer instance.
type LifecycleOption func(*LifecycleServer)

// WithLifecycleConfigure sets the callback executed on Configure RPC.
func WithLifecycleConfigure(fn func(ctx context.Context, values map[string]string, secrets Secrets) error) LifecycleOption {
	return func(s *LifecycleServer) {
		s.onConfigure = fn
	}
}

// WithLifecycleUpdateConfig adapts a configuration map updater to the Configure RPC.
func WithLifecycleUpdateConfig(fn func(values map[string]string, secrets Secrets)) LifecycleOption {
	return func(s *LifecycleServer) {
		if fn == nil {
			s.onConfigure = nil
			return
		}
		s.onConfigure = func(_ context.Context, values map[string]string, secrets Secrets) error {
			fn(values, secrets)
			return nil
		}
	}
}

// WithLifecycleShutdown sets what the Shutdown RPC runs before it signals termination. The RPC
// signals termination even when fn fails, and answers the failure.
func WithLifecycleShutdown(fn func() error) LifecycleOption {
	return func(s *LifecycleServer) {
		s.onShutdown = fn
	}
}

// LifecycleServer holds the lifecycle state of a plugin: its configuration and its shutdown signal.
// Server serves it to the host as the plugin lifecycle service.
type LifecycleServer struct {
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
	mu           sync.RWMutex
	config       map[string]string
	onConfigure  func(ctx context.Context, values map[string]string, secrets Secrets) error
	onShutdown   func() error
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
	if s.onShutdown != nil {
		closeErr = s.onShutdown()
	}
	s.TriggerShutdown()
	return closeErr
}

func (s *LifecycleServer) configureWith(ctx context.Context, values map[string]string, secrets Secrets) error {
	s.mu.Lock()
	if s.config == nil {
		s.config = make(map[string]string)
	}
	maps.Copy(s.config, values)
	callback := s.onConfigure
	s.mu.Unlock()

	if callback != nil {
		return callback(ctx, values, secrets)
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
func (s *LifecycleServer) SetConfigure(fn func(ctx context.Context, values map[string]string, secrets Secrets) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onConfigure = fn
}

// SetUpdateConfig registers a simplified configuration updater callback.
func (s *LifecycleServer) SetUpdateConfig(fn func(values map[string]string, secrets Secrets)) {
	s.SetConfigure(func(_ context.Context, values map[string]string, secrets Secrets) error {
		fn(values, secrets)
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

// CheckStatus returns the status a ping answers with.
func (s *LifecycleServer) CheckStatus(context.Context) (string, error) {
	return statusOK, nil
}

// ApplyConfig stores the configuration values and runs the configure callback with the values and the secrets.
func (s *LifecycleServer) ApplyConfig(ctx context.Context, values map[string]string, secrets Secrets) (bool, error) {
	if err := s.configureWith(ctx, values, secrets); err != nil {
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
