package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func (s *subscriptionStub) ListProducts(_ context.Context, req *connect.Request[syncv1.ListProductsRequest]) (*connect.Response[syncv1.ListProductsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.ListProductsResponse{Products: nil}), nil
}

func (s *subscriptionStub) GetEntitlements(_ context.Context, req *connect.Request[syncv1.GetEntitlementsRequest]) (*connect.Response[syncv1.GetEntitlementsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetEntitlementsResponse{Entitlements: nil}), nil
}

func (s *subscriptionStub) CreatePortalSession(_ context.Context, req *connect.Request[syncv1.CreatePortalSessionRequest]) (*connect.Response[syncv1.CreatePortalSessionResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.CreatePortalSessionResponse{PortalUrl: s.portal}), nil
}

func TestGetSubscription_ReadsTheTrialMark(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.state = &syncv1.GetSubscriptionResponse{Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, Plan: "plus", PeriodEndAt: 1_800_000_000, IsTrial: true}

	got, err := newSubscriptionClient(address).GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !got.IsTrial || !got.PeriodEnd.Equal(time.Unix(1_800_000_000, 0)) {
		t.Errorf("subscription = %+v, want a trial that ends at 1800000000", got)
	}
}

func TestCreatePortalSession_ReturnsTheAddress(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.portal = "https://portal.example.test/session/1"

	got, err := newSubscriptionClient(address).CreatePortalSession(t.Context())

	if err != nil || got != "https://portal.example.test/session/1" {
		t.Errorf("address = %q, error = %v, want the portal address", got, err)
	}
}

func TestCreatePortalSession_RefusesAnAddressThatIsNotHTTPSOrLoopbackHTTP(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.portal = "http://portal.example.test/session/1"

	if got, err := newSubscriptionClient(address).CreatePortalSession(t.Context()); err == nil {
		t.Errorf("address = %q, want an error for plain http on a public host", got)
	}
}

func TestCreatePortalSession_MapsFailedPreconditionToErrNoActiveSubscription(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeFailedPrecondition, errors.New("never paid"))

	_, err := newSubscriptionClient(address).CreatePortalSession(t.Context())

	if !errors.Is(err, syncprotocol.ErrNoActiveSubscription) {
		t.Errorf("error = %v, want ErrNoActiveSubscription", err)
	}
}
