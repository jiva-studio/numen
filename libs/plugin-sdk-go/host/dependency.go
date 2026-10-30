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

// DependencyState is the installation state of a dependency.
type DependencyState int32

const (
	DependencyStateUnspecified  DependencyState = 0
	DependencyStateMissing      DependencyState = 1
	DependencyStateProvisioning DependencyState = 2
	DependencyStateReady        DependencyState = 3
	DependencyStateFailed       DependencyState = 4
)

// DependencyStatus reports the state of a dependency.
type DependencyStatus struct {
	ID              string
	State           DependencyState
	ProgressPercent float64
	Message         string
	Error           string
}

func (s *DependencyStatus) GetId() string {
	if s == nil {
		return ""
	}
	return s.ID
}

func (s *DependencyStatus) GetState() DependencyState {
	if s == nil {
		return DependencyStateUnspecified
	}
	return s.State
}

// DependencyRequirement describes a dependency to provision.
type DependencyRequirement struct {
	ID string
}

func readDependencyStatus(st *v1.DependencyStatus) *DependencyStatus {
	if st == nil {
		return nil
	}
	return &DependencyStatus{
		ID:              st.GetId(),
		State:           DependencyState(st.GetState()),
		ProgressPercent: st.GetProgressPercent(),
		Message:         st.GetMessage(),
		Error:           st.GetError(),
	}
}

func writeDependencyRequirement(req *DependencyRequirement) *v1.DependencyRequirement {
	if req == nil {
		return nil
	}
	return &v1.DependencyRequirement{Id: req.ID}
}

// GetStatus answers the state of the dependency with the id.
func (c *DependencyClient) GetStatus(ctx context.Context, id string) (*DependencyStatus, error) {
	res, err := c.rpc.GetDependencyStatus(ctx, connect.NewRequest(&v1.GetDependencyStatusRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return readDependencyStatus(res.Msg.GetStatus()), nil
}

// Provision asks the host to install the requirement and yields each status the host streams, ending
// with READY or FAILED. A broken stream yields its error once and stops.
func (c *DependencyClient) Provision(ctx context.Context, requirement *DependencyRequirement) iter.Seq2[*DependencyStatus, error] {
	return func(yield func(*DependencyStatus, error) bool) {
		stream, err := c.rpc.ProvisionDependency(ctx, connect.NewRequest(&v1.ProvisionDependencyRequest{Requirement: writeDependencyRequirement(requirement)}))
		if err != nil {
			yield(nil, err)
			return
		}
		defer closeStream(stream)
		for stream.Receive() {
			if !yield(readDependencyStatus(stream.Msg().GetStatus()), nil) {
				return
			}
		}
		if err := stream.Err(); err != nil {
			yield(nil, err)
		}
	}
}

// List answers the state of every dependency the host knows.
func (c *DependencyClient) List(ctx context.Context) ([]*DependencyStatus, error) {
	res, err := c.rpc.ListDependencies(ctx, connect.NewRequest(&v1.ListDependenciesRequest{}))
	if err != nil {
		return nil, err
	}
	out := make([]*DependencyStatus, len(res.Msg.GetStatuses()))
	for i, s := range res.Msg.GetStatuses() {
		out[i] = readDependencyStatus(s)
	}
	return out, nil
}
