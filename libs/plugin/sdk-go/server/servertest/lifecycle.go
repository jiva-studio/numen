// Package servertest holds the client plugin tests call the lifecycle service of a plugin with.
package servertest

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// LifecycleClient calls the lifecycle service of a plugin.
type LifecycleClient struct {
	rpc numenv1connect.PluginLifecycleServiceClient
}

// NewLifecycleClient returns a client that presents token on every call, as the host does.
func NewLifecycleClient(httpClient connect.HTTPClient, baseURL, token string) *LifecycleClient {
	rpc := numenv1connect.NewPluginLifecycleServiceClient(httpClient, baseURL, connect.WithInterceptors(hostcall.Bearer(token)))
	return &LifecycleClient{rpc: rpc}
}

// Ping asks the plugin for its status and protocol version.
func (c *LifecycleClient) Ping(ctx context.Context) (status string, protocol int32, err error) {
	res, err := c.rpc.Ping(ctx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		return "", 0, err
	}
	return res.Msg.GetStatus(), res.Msg.GetProtocol(), nil
}

// Configure sends the configuration values to the plugin and reports whether it accepted them.
func (c *LifecycleClient) Configure(ctx context.Context, values map[string]string) (bool, error) {
	res, err := c.rpc.Configure(ctx, connect.NewRequest(&v1.ConfigureRequest{Values: values}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetSuccess(), nil
}

// Shutdown asks the plugin to terminate and reports whether it accepted.
func (c *LifecycleClient) Shutdown(ctx context.Context) (bool, error) {
	res, err := c.rpc.Shutdown(ctx, connect.NewRequest(&v1.ShutdownRequest{}))
	if err != nil {
		return false, err
	}
	return res.Msg.GetSuccess(), nil
}
