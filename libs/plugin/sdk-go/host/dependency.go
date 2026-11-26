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
	return getFetchedPath(res.Msg.GetPath(), res.Msg.GetError(), res.Msg.GetDetail())
}

// FetchDependency returns the path of the model that the manifest of the plugin declares under id.
// The host fetches it first when it is not on this machine.
func (c *DependencyClient) FetchDependency(ctx context.Context, id string) (string, error) {
	res, err := c.rpc.FetchDependency(ctx, connect.NewRequest(&v1.FetchDependencyRequest{Id: id}))
	if err != nil {
		return "", err
	}
	return getFetchedPath(res.Msg.GetPath(), res.Msg.GetError(), res.Msg.GetDetail())
}

func getFetchedPath(path string, code v1.ErrorCode, detail string) (string, error) {
	if code == v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		return path, nil
	}
	if detail != "" {
		return "", errors.New(detail)
	}
	return "", fail(code)
}
