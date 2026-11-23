package host

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/internal/transport"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// DependencyClient resolves shared model files through the host.
type DependencyClient struct {
	rpc numenv1connect.HostDependencyServiceClient
}

func NewDependencyClient(endpoint hostcall.Endpoint) *DependencyClient {
	return &DependencyClient{rpc: numenv1connect.NewHostDependencyServiceClient(transport.GetHTTPClient(), endpoint.BaseURL, transport.BuildOptions(endpoint)...)}
}

func (c *DependencyClient) FetchModelFile(ctx context.Context, address, digest string) (string, error) {
	res, err := c.rpc.FetchModelFile(ctx, connect.NewRequest(&v1.FetchModelFileRequest{Url: address, Sha256: digest}))
	if err != nil {
		return "", err
	}
	if message := res.Msg.GetError(); message != "" {
		return "", errors.New(message)
	}
	return res.Msg.GetPath(), nil
}
