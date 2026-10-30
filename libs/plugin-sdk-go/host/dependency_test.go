package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

type provisioning struct {
	numenv1connect.UnimplementedDependencyServiceHandler
	provisioned *v1.DependencyRequirement
}

func (p *provisioning) GetDependencyStatus(_ context.Context, req *connect.Request[v1.GetDependencyStatusRequest]) (*connect.Response[v1.GetDependencyStatusResponse], error) {
	return connect.NewResponse(&v1.GetDependencyStatusResponse{Status: &v1.DependencyStatus{Id: req.Msg.GetId(), State: v1.DependencyState_DEPENDENCY_STATE_MISSING}}), nil
}

func (p *provisioning) ProvisionDependency(_ context.Context, req *connect.Request[v1.ProvisionDependencyRequest], stream *connect.ServerStream[v1.ProvisionDependencyResponse]) error {
	p.provisioned = req.Msg.GetRequirement()
	id := p.provisioned.GetId()
	if id == "broken" {
		return connect.NewError(connect.CodeInternal, errors.New("disk"))
	}
	if err := stream.Send(&v1.ProvisionDependencyResponse{Status: &v1.DependencyStatus{Id: id, State: v1.DependencyState_DEPENDENCY_STATE_PROVISIONING, ProgressPercent: 50}}); err != nil {
		return err
	}
	return stream.Send(&v1.ProvisionDependencyResponse{Status: &v1.DependencyStatus{Id: id, State: v1.DependencyState_DEPENDENCY_STATE_READY}})
}

func (p *provisioning) ListDependencies(context.Context, *connect.Request[v1.ListDependenciesRequest]) (*connect.Response[v1.ListDependenciesResponse], error) {
	return connect.NewResponse(&v1.ListDependenciesResponse{Statuses: []*v1.DependencyStatus{{Id: "a"}, {Id: "b"}}}), nil
}

func newDependencyClient(t *testing.T, p *provisioning) *host.DependencyClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewDependencyServiceHandler(p))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return host.NewDependencyClient(hostcall.Endpoint{BaseURL: srv.URL, Token: "tok"})
}

func TestDependencyClientGetStatusAndList(t *testing.T) {
	c := newDependencyClient(t, &provisioning{})
	st, err := c.GetStatus(t.Context(), "whisper")
	if err != nil || st.GetID() != "whisper" || st.GetState() != host.DependencyStateMissing {
		t.Fatalf("GetStatus = %v, %v", st, err)
	}
	list, err := c.List(t.Context())
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %v, %v", list, err)
	}
}

func TestDependencyClientProvisionStreamsUntilReady(t *testing.T) {
	p := &provisioning{}
	c := newDependencyClient(t, p)
	var states []host.DependencyState
	for st, err := range c.Provision(t.Context(), &host.DependencyRequirement{ID: "whisper"}) {
		if err != nil {
			t.Fatal(err)
		}
		states = append(states, st.GetState())
	}
	if len(states) != 2 || states[1] != host.DependencyStateReady || p.provisioned.GetId() != "whisper" {
		t.Fatalf("states = %v", states)
	}
}

func TestDependencyClientProvisionYieldsStreamError(t *testing.T) {
	c := newDependencyClient(t, &provisioning{})
	var got error
	for _, err := range c.Provision(t.Context(), &host.DependencyRequirement{ID: "broken"}) {
		got = err
	}
	if connect.CodeOf(got) != connect.CodeInternal {
		t.Fatalf("err = %v", got)
	}
}
