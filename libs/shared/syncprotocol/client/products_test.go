package client_test

import (
	"context"
	"errors"
	"slices"
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
	return connect.NewResponse(&syncv1.ListProductsResponse{Products: s.products}), nil
}

func (s *subscriptionStub) GetEntitlements(_ context.Context, req *connect.Request[syncv1.GetEntitlementsRequest]) (*connect.Response[syncv1.GetEntitlementsResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.GetEntitlementsResponse{Entitlements: s.entitlements}), nil
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

func TestListProducts_ReturnsTheProductsInTheOrderOfTheServer(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.products = []*syncv1.Product{
		{Id: "prod_link", Slug: "link", Name: "Link", PriceCents: 2000, Currency: "USD", OneTime: true},
		{Id: "prod_pack", Slug: "pack", Name: "Pack", PriceCents: 300, Currency: "USD"},
	}

	got, err := newSubscriptionClient(address).ListProducts(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []syncprotocol.Product{
		{ID: "prod_link", Slug: "link", Name: "Link", PriceCents: 2000, Currency: "USD", OneTime: true},
		{ID: "prod_pack", Slug: "pack", Name: "Pack", PriceCents: 300, Currency: "USD"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("products = %+v, want %+v", got, want)
	}
}

func TestListProducts_SendsTheSessionAndTheVersion(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)

	if _, err := newSubscriptionClient(address).ListProducts(t.Context()); err != nil {
		t.Fatal(err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 1 || stub.headers[0].Get("Authorization") != "Bearer token-of-the-session" || stub.headers[0].Get(syncprotocol.HeaderVersion) == "" {
		t.Errorf("headers = %v, want the bearer token and the protocol version", stub.headers)
	}
}

func TestGetEntitlements_ReturnsTheProductsWithTheEndThePhaseAndTheQuota(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.entitlements = []*syncv1.Entitlement{
		{Product: "link", ValidUntil: 4_900_000_000, Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE},
		{Product: "sync", ValidUntil: 1_800_000_000, Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, QuotaBytes: 100 << 30, IsTrial: true},
	}

	got, err := newSubscriptionClient(address).GetEntitlements(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []syncprotocol.Entitlement{
		{Product: "link", ValidUntil: time.Unix(4_900_000_000, 0), Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE},
		{Product: "sync", ValidUntil: time.Unix(1_800_000_000, 0), Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, QuotaBytes: 100 << 30, IsTrial: true},
	}
	if !slices.EqualFunc(got, want, func(a, b syncprotocol.Entitlement) bool {
		return a.Product == b.Product && a.ValidUntil.Equal(b.ValidUntil) && a.Phase == b.Phase && a.QuotaBytes == b.QuotaBytes && a.IsTrial == b.IsTrial
	}) {
		t.Errorf("entitlements = %+v, want %+v", got, want)
	}
}

func TestGetEntitlements_ReturnsNoneForAnAccountThatOwnsNone(t *testing.T) {
	t.Parallel()
	_, address := serveSubscriptionStub(t, syncprotocol.Version)

	got, err := newSubscriptionClient(address).GetEntitlements(t.Context())

	if err != nil || len(got) != 0 {
		t.Errorf("entitlements = %+v, error = %v, want none", got, err)
	}
}

func TestGetEntitlements_MapsUnimplementedToErrBillingUnavailable(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeUnimplemented, errors.New("no billing"))

	_, err := newSubscriptionClient(address).GetEntitlements(t.Context())

	if !errors.Is(err, syncprotocol.ErrBillingUnavailable) {
		t.Errorf("error = %v, want ErrBillingUnavailable", err)
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
