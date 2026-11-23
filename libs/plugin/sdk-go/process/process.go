// Package process lets a plugin tell the host that its process has finished its work. The host stops a
// released process at once where no call into it is open, and else as soon as the last call ends. A
// call the host makes into the plugin after the release keeps the process for the idle limit again.
package process

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/internal/transport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Client releases the process of the plugin.
type Client struct {
	rpc numenv1connect.HostProcessServiceClient
}

// NewClient returns the client of the host at the endpoint, sending its calls with the default HTTP client.
func NewClient(endpoint hostcall.Endpoint) *Client {
	return newClientOver(transport.GetHTTPClient(), endpoint)
}

func newClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewHostProcessServiceClient(httpClient, endpoint.BaseURL, transport.BuildOptions(endpoint)...)}
}

// NewClientFromEnv returns the client of the host the process was started for. It returns
// hostcall.ErrNoHost where the environment names none.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.ReadEnv()
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
