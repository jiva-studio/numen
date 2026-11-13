package providerbridge

import (
	"context"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// RunningHost hands out a plugin whose process runs, with a question lease on it, without starting it.
type RunningHost interface {
	AcquireRunningForQuestion(pluginID string) (transport.Endpoint, bool)
	ReleaseQuestion(pluginID string)
}

// RunningSource opens the provider of a plugin only while its process runs, for a question the core asks
// it for itself.
type RunningSource struct {
	host    RunningHost
	clients transport.ClientSource
}

// NewRunningSource returns the source over the running plugins of host, which reaches each through the
// client clients keeps for its endpoint.
func NewRunningSource(host RunningHost, clients transport.ClientSource) *RunningSource {
	return &RunningSource{host: host, clients: clients}
}

// OpenRunningProvider returns the provider of the plugin with the function that releases the plugin, and
// false where the process of the plugin does not run. The plugin is never started.
func (s *RunningSource) OpenRunningProvider(_ context.Context, pluginID string) (connectionport.Provider, func(), bool) {
	endpoint, isRunning := s.host.AcquireRunningForQuestion(pluginID)
	if !isRunning {
		return nil, nil, false
	}
	interceptor := connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token))
	client := numenv1connect.NewProviderServiceClient(s.clients.GetClient(endpoint), transport.GetBaseURL(endpoint), interceptor)
	var once sync.Once
	return &provider{client: client}, func() { once.Do(func() { s.host.ReleaseQuestion(pluginID) }) }, true
}
