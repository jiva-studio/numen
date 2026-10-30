package server

import (
	"context"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

const ProtocolVersion = 1

type PingRequest = v1.PingRequest

type PingResponse = v1.PingResponse

type ConfigureRequest = v1.ConfigureRequest

type ConfigureResponse = v1.ConfigureResponse

type ShutdownRequest = v1.ShutdownRequest

type ShutdownResponse = v1.ShutdownResponse

type PluginLifecycleServiceHandler = numenv1connect.PluginLifecycleServiceHandler

type PluginLifecycleServiceClient = numenv1connect.PluginLifecycleServiceClient

// NewPluginLifecycleServiceHandler builds an HTTP handler for the lifecycle service.
func NewPluginLifecycleServiceHandler(svc PluginLifecycleServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewPluginLifecycleServiceHandler(svc, opts...)
}

// NewPluginLifecycleServiceClient constructs a client for the lifecycle service.
func NewPluginLifecycleServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) PluginLifecycleServiceClient {
	return numenv1connect.NewPluginLifecycleServiceClient(httpClient, baseURL, opts...)
}

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

// WithLifecycleShutdown sets what the Shutdown RPC runs before it signals termination.
func WithLifecycleShutdown(fn func()) LifecycleOption {
	return func(s *LifecycleServer) {
		s.onShutdown = fn
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
	onShutdown   func()
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

func (s *LifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{
		Status:   "ok",
		Protocol: ProtocolVersion,
	}), nil
}

// Shutdown signals termination to the server runner without blocking.
func (s *LifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	if s.onShutdown != nil {
		s.onShutdown()
	}
	s.TriggerShutdown()
	return connect.NewResponse(&v1.ShutdownResponse{
		Success: true,
	}), nil
}

// Configure persists configuration values and invokes the registered configure callback.
func (s *LifecycleServer) Configure(
	ctx context.Context,
	req *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
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

	return connect.NewResponse(&v1.ConfigureResponse{
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

// CheckStatus executes a Ping check and returns the status string.
func (s *LifecycleServer) CheckStatus(ctx context.Context) (string, error) {
	res, err := s.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		return "", err
	}
	return res.Msg.GetStatus(), nil
}

// ApplyConfig applies configuration values through Configure.
func (s *LifecycleServer) ApplyConfig(ctx context.Context, values map[string]string) (bool, error) {
	res, err := s.Configure(ctx, connect.NewRequest(&v1.ConfigureRequest{Values: values}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetSuccess(), nil
}

// RequestShutdown triggers shutdown through Shutdown.
func (s *LifecycleServer) RequestShutdown(ctx context.Context) (bool, error) {
	res, err := s.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetSuccess(), nil
}
