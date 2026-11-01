package server

import (
	"context"
	"sync"
)

// ProtocolVersion is the protocol version the SDK uses.
const ProtocolVersion = 1

// statusOK is the status a healthy plugin answers a ping with.
const statusOK = "ok"

// LifecycleOption configures a LifecycleServer instance.
type LifecycleOption func(*LifecycleServer)

// WithLifecycleConfigure sets the callback executed on Configure RPC.
func WithLifecycleConfigure(fn func(ctx context.Context, values map[string]string) error) LifecycleOption {
	return func(s *LifecycleServer) {
		s.onConfigure = fn
	}
}

// WithLifecycleUpdateConfig adapts a configuration map updater to the Configure RPC.
func WithLifecycleUpdateConfig(fn func(values map[string]string)) LifecycleOption {
	return func(s *LifecycleServer) {
		if fn == nil {
			s.onConfigure = nil
			return
		}
		s.onConfigure = func(_ context.Context, values map[string]string) error {
			fn(values)
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
	onConfigure  func(ctx context.Context, values map[string]string) error
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

// shutdown runs the shutdown callback and signals termination. The signal is sent even when the
// callback fails.
func (s *LifecycleServer) shutdown() error {
	var closeErr error
	if s.onShutdown != nil {
		closeErr = s.onShutdown()
	}
	s.TriggerShutdown()
	return closeErr
}

// configure stores the values and invokes the registered configure callback.
func (s *LifecycleServer) configure(ctx context.Context, values map[string]string) error {
	s.mu.Lock()
	if s.config == nil {
		s.config = make(map[string]string)
	}
	for k, v := range values {
		s.config[k] = v
	}
	callback := s.onConfigure
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
	s.onConfigure = fn
}

// SetUpdateConfig registers a simplified configuration updater callback.
func (s *LifecycleServer) SetUpdateConfig(fn func(values map[string]string)) {
	s.SetConfigure(func(_ context.Context, values map[string]string) error {
		fn(values)
		return nil
	})
}

// GetConfig returns the configuration value for a key.
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
	for k, v := range s.config {
		res[k] = v
	}
	return res
}

// CheckStatus returns the status a ping answers with.
func (s *LifecycleServer) CheckStatus(context.Context) (string, error) {
	return statusOK, nil
}

// ApplyConfig stores the configuration values and runs the configure callback.
func (s *LifecycleServer) ApplyConfig(ctx context.Context, values map[string]string) (bool, error) {
	if err := s.configure(ctx, values); err != nil {
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
