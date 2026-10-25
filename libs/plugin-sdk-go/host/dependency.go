package host

import (
	"context"
	"iter"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// DependencyClient asks the host about the dependencies plugins declare, through DependencyService.
type DependencyClient struct {
	rpc numenv1connect.DependencyServiceClient
}

// NewDependencyClientFromEnv reaches the host the process was started for. It returns
// hostcall.ErrNoHost where the host named none.
func NewDependencyClientFromEnv() (*DependencyClient, error) {
	addr, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewDependencyClient(addr), nil
}

// NewDependencyClient reaches the host at the endpoint.
func NewDependencyClient(endpoint hostcall.Endpoint) *DependencyClient {
	return NewDependencyClientOver(hostcall.HTTPClient(), endpoint)
}

// NewDependencyClientOver reaches the host at the endpoint through the HTTP client given.
func NewDependencyClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *DependencyClient {
	return &DependencyClient{rpc: numenv1connect.NewDependencyServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

// GetStatus answers the state of the dependency with the id.
func (c *DependencyClient) GetStatus(ctx context.Context, id string) (*v1.DependencyStatus, error) {
	res, err := c.rpc.GetDependencyStatus(ctx, connect.NewRequest(&v1.GetDependencyStatusRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return res.Msg.GetStatus(), nil
}

// Provision asks the host to install the requirement and yields each status the host streams, ending
// with READY or FAILED. A broken stream yields its error once and stops.
func (c *DependencyClient) Provision(ctx context.Context, requirement *v1.DependencyRequirement) iter.Seq2[*v1.DependencyStatus, error] {
	return func(yield func(*v1.DependencyStatus, error) bool) {
		stream, err := c.rpc.ProvisionDependency(ctx, connect.NewRequest(&v1.ProvisionDependencyRequest{Requirement: requirement}))
		if err != nil {
			yield(nil, err)
			return
		}
		defer func() { _ = stream.Close() }()
		for stream.Receive() {
			if !yield(stream.Msg().GetStatus(), nil) {
				return
			}
		}
		if err := stream.Err(); err != nil {
			yield(nil, err)
		}
	}
}

// List answers the state of every dependency the host knows.
func (c *DependencyClient) List(ctx context.Context) ([]*v1.DependencyStatus, error) {
	res, err := c.rpc.ListDependencies(ctx, connect.NewRequest(&v1.ListDependenciesRequest{}))
	if err != nil {
		return nil, err
	}
	return res.Msg.GetStatuses(), nil
}
