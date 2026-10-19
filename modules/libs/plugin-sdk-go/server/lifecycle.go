package server

import (
	"context"
	"sync"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// ProtocolVersion is the plugin protocol this SDK speaks. Ping reports it to the host.
const ProtocolVersion = 1

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

// LifecycleServer implements numen.v1.PluginLifecycleService.
type LifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownCh   chan struct{}
	shutdownOnce sync.Once
	mu           sync.RWMutex
	config       map[string]string
	onConfigure  func(ctx context.Context, values map[string]string) error
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

// Ping answers health check probes from the plugin host with the protocol version the plugin speaks.
func (s *LifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[numenv1.PingRequest],
) (*connect.Response[numenv1.PingResponse], error) {
	return connect.NewResponse(&numenv1.PingResponse{
		Status:   "ok",
		Protocol: ProtocolVersion,
	}), nil
}

// Shutdown signals termination to the server runner without blocking.
func (s *LifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[numenv1.ShutdownRequest],
) (*connect.Response[numenv1.ShutdownResponse], error) {
	s.TriggerShutdown()
	return connect.NewResponse(&numenv1.ShutdownResponse{
		Success: true,
	}), nil
}

// Configure persists configuration values and invokes the registered configure callback.
func (s *LifecycleServer) Configure(
	ctx context.Context,
	req *connect.Request[numenv1.ConfigureRequest],
) (*connect.Response[numenv1.ConfigureResponse], error) {
	values := req.Msg.GetValues()

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
		if err := callback(ctx, values); err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&numenv1.ConfigureResponse{
		Success: true,
	}), nil
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
