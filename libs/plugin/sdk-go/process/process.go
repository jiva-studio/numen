// Package process lets a plugin tell the host that its process has finished its work. The host stops a
// released process at once where no call into it is open, and else at its first idle check after the
// last call ends. A call the host makes into the plugin after the release keeps the process for the
// idle limit again.
package process

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// Client releases the process of the plugin.
type Client struct {
	rpc numenv1connect.PluginServiceClient
}

// NewClient returns the client of the host at the endpoint, sending its calls with hostcall.HTTPClient.
func NewClient(endpoint hostcall.Endpoint) *Client {
	return NewClientOver(hostcall.HTTPClient(), endpoint)
}

// NewClientOver returns the client of the host at the endpoint, sending its calls with httpClient.
func NewClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewPluginServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

// NewClientFromEnv returns the client of the host the process was started for. It returns
// hostcall.ErrNoHost where the environment names none.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(endpoint), nil
}

// Release tells the host that the process has finished its work. The answer can arrive after the host
// has stopped the process, or never where the stop ends the process first.
func (c *Client) Release(ctx context.Context) error {
	_, err := c.rpc.ReleaseProcess(ctx, connect.NewRequest(&v1.ReleaseProcessRequest{}))
	return err
}
