package client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type subscriptionStub struct {
	syncv1connect.UnimplementedSubscriptionServiceHandler

	mu       sync.Mutex
	headers  []http.Header
	fail     error
	state    *syncv1.GetSubscriptionResponse
	checkout string
	plans    []*syncv1.Plan
	asked    []string

	products     []*syncv1.Product
	entitlements []*syncv1.Entitlement
	portal       string
}

func (s *subscriptionStub) record(header http.Header) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.headers = append(s.headers, header.Clone())
	return s.fail
}

func (s *subscriptionStub) GetSubscription(_ context.Context, req *connect.Request[syncv1.GetSubscriptionRequest]) (*connect.Response[syncv1.GetSubscriptionResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(s.state), nil
}

func (s *subscriptionStub) CreateCheckout(_ context.Context, req *connect.Request[syncv1.CreateCheckoutRequest]) (*connect.Response[syncv1.CreateCheckoutResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.asked = append(s.asked, req.Msg.GetPlan())
	s.mu.Unlock()
	return connect.NewResponse(&syncv1.CreateCheckoutResponse{CheckoutUrl: s.checkout}), nil
}

func serveSubscriptionStub(tb testing.TB, serverVersion uint32) (*subscriptionStub, string) {
	tb.Helper()
	stub := &subscriptionStub{state: &syncv1.GetSubscriptionResponse{Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE}}
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewSubscriptionServiceHandler(stub, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(serverVersion))))
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return stub, server.URL
}

func newSubscriptionClient(address string) *syncclient.SubscriptionClient {
	return syncclient.NewSubscriptionClient(address, http.DefaultClient, func() string { return "token-of-the-session" })
}

func TestSubscriptionClient_SendsTheVersionAndTheBearerOfTheTokenInEachRequest(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.checkout = "https://pay.example/checkout/1"
	client := newSubscriptionClient(address)

	if _, err := client.GetSubscription(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateCheckout(t.Context(), "plus"); err != nil {
		t.Fatal(err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 2 {
		t.Fatalf("the server saw %d requests, want 2", len(stub.headers))
	}
	for i, header := range stub.headers {
		if got := header.Get("Authorization"); got != "Bearer token-of-the-session" {
			t.Errorf("request %d: Authorization = %q, want the bearer of the token", i, got)
		}
		if got := header.Get(syncprotocol.HeaderVersion); got != "1" {
			t.Errorf("request %d: %s = %q, want 1", i, syncprotocol.HeaderVersion, got)
		}
	}
}

func TestGetSubscription_ReadsThePhaseAndTheTimesOfTheResponse(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.state = &syncv1.GetSubscriptionResponse{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, PeriodEndAt: 1_700_000_000, WritesEndAt: 1_700_604_800, DeleteAt: 1_705_788_800,
	}

	got, err := newSubscriptionClient(address).GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE {
		t.Errorf("phase = %v, want grace", got.Phase)
	}
	if want := time.Unix(1_700_000_000, 0); !got.PeriodEnd.Equal(want) {
		t.Errorf("PeriodEnd = %v, want %v", got.PeriodEnd, want)
	}
	if want := time.Unix(1_700_604_800, 0); !got.WritesEnd.Equal(want) {
		t.Errorf("WritesEnd = %v, want %v", got.WritesEnd, want)
	}
	if want := time.Unix(1_705_788_800, 0); !got.DeleteAt.Equal(want) {
		t.Errorf("DeleteAt = %v, want %v", got.DeleteAt, want)
	}
}

func TestGetSubscription_ReturnsZeroTimesWhenTheServerSentNone(t *testing.T) {
	t.Parallel()
	_, address := serveSubscriptionStub(t, syncprotocol.Version)

	got, err := newSubscriptionClient(address).GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE || !got.PeriodEnd.IsZero() || !got.WritesEnd.IsZero() || !got.DeleteAt.IsZero() {
		t.Errorf("subscription = %+v, want phase none and zero times", got)
	}
}

func TestGetSubscription_RefusesAnUnspecifiedPhase(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.state = &syncv1.GetSubscriptionResponse{Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_UNSPECIFIED}

	if _, err := newSubscriptionClient(address).GetSubscription(t.Context()); err == nil {
		t.Error("GetSubscription of an unspecified phase succeeded, want an error")
	}
}

func TestCreateCheckout_ReturnsTheUrlOfTheServer(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.checkout = "https://pay.example/checkout/abc"

	got, err := newSubscriptionClient(address).CreateCheckout(t.Context(), "plus")

	if err != nil || got != "https://pay.example/checkout/abc" {
		t.Errorf("url = %q, error = %v, want the url of the server", got, err)
	}
}

func TestCreateCheckout_AcceptsHttpOnALoopbackHost(t *testing.T) {
	t.Parallel()
	for _, url := range []string{"http://127.0.0.1:8080/pay/1", "http://localhost:8080/pay/1", "http://[::1]:8080/pay/1"} {
		stub, address := serveSubscriptionStub(t, syncprotocol.Version)
		stub.checkout = url

		got, err := newSubscriptionClient(address).CreateCheckout(t.Context(), "plus")

		if err != nil || got != url {
			t.Errorf("url %q: got %q, error = %v, want the url accepted", url, got, err)
		}
	}
}

func TestCreateCheckout_RefusesAnAddressThatABrowserMustNotOpen(t *testing.T) {
	t.Parallel()
	for name, url := range map[string]string{
		"empty":              "",
		"http remote":        "http://pay.example/checkout/1",
		"file":               "file:///etc/passwd",
		"custom scheme":      "numen://open",
		"javascript":         "javascript:alert(1)",
		"no scheme":          "pay.example/checkout/1",
		"not a url":          "https://%zz",
		"https without host": "https:///checkout",
	} {
		stub, address := serveSubscriptionStub(t, syncprotocol.Version)
		stub.checkout = url

		got, err := newSubscriptionClient(address).CreateCheckout(t.Context(), "plus")

		if err == nil {
			t.Errorf("%s: CreateCheckout returned %q, want an error", name, got)
		}
	}
}

func TestSubscriptionClient_MapsServerFailuresToTheErrorsOfThePackage(t *testing.T) {
	t.Parallel()
	for name, one := range map[string]struct {
		err  error
		want error
	}{
		"already exists":  {err: connect.NewError(connect.CodeAlreadyExists, errors.New("active")), want: syncprotocol.ErrAlreadySubscribed},
		"unimplemented":   {err: connect.NewError(connect.CodeUnimplemented, errors.New("no billing")), want: syncprotocol.ErrBillingUnavailable},
		"unauthenticated": {err: connect.NewError(connect.CodeUnauthenticated, errors.New("no")), want: syncprotocol.ErrUnauthenticated},
		"unavailable":     {err: connect.NewError(connect.CodeUnavailable, errors.New("down")), want: syncprotocol.ErrUnreachable},
		"deadline":        {err: connect.NewError(connect.CodeDeadlineExceeded, errors.New("slow")), want: syncprotocol.ErrUnreachable},
	} {
		stub, address := serveSubscriptionStub(t, syncprotocol.Version)
		stub.fail = one.err
		stub.checkout = "https://pay.example/checkout/1"
		client := newSubscriptionClient(address)

		_, getErr := client.GetSubscription(t.Context())
		_, checkoutErr := client.CreateCheckout(t.Context(), "plus")

		if !errors.Is(getErr, one.want) {
			t.Errorf("GetSubscription of %s: error = %v, want %v", name, getErr, one.want)
		}
		if !errors.Is(checkoutErr, one.want) {
			t.Errorf("CreateCheckout of %s: error = %v, want %v", name, checkoutErr, one.want)
		}
	}
}

func TestCreateCheckout_ReportsAnActiveSubscriptionNotAsAnExistingVault(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeAlreadyExists, errors.New("active"))

	_, err := newSubscriptionClient(address).CreateCheckout(t.Context(), "plus")

	if errors.Is(err, syncprotocol.ErrVaultExists) {
		t.Errorf("error = %v, want ErrAlreadySubscribed and not ErrVaultExists", err)
	}
}

func TestSubscriptionClient_ReportsAServerOfAnotherVersionAsUpdateRequired(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version+1)
	client := newSubscriptionClient(address)

	_, getErr := client.GetSubscription(t.Context())
	_, checkoutErr := client.CreateCheckout(t.Context(), "plus")

	if !errors.Is(getErr, syncprotocol.ErrUpdateRequired) || !errors.Is(checkoutErr, syncprotocol.ErrUpdateRequired) {
		t.Errorf("errors = %v and %v, want ErrUpdateRequired", getErr, checkoutErr)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 0 {
		t.Errorf("the server ran %d calls, want none", len(stub.headers))
	}
}

func (s *subscriptionStub) ListPlans(_ context.Context, req *connect.Request[syncv1.ListPlansRequest]) (*connect.Response[syncv1.ListPlansResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&syncv1.ListPlansResponse{Plans: s.plans}), nil
}

func (s *subscriptionStub) ChangePlan(_ context.Context, req *connect.Request[syncv1.ChangePlanRequest]) (*connect.Response[syncv1.ChangePlanResponse], error) {
	if err := s.record(req.Header()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.asked = append(s.asked, req.Msg.GetPlan())
	s.mu.Unlock()
	return connect.NewResponse(&syncv1.ChangePlanResponse{Subscription: s.state}), nil
}

func TestGetSubscription_ReadsThePlanTheQuotaAndTheUsedBytes(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.state = &syncv1.GetSubscriptionResponse{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, Plan: "plus", QuotaBytes: 100 << 30, UsedBytes: 2 << 30,
	}

	got, err := newSubscriptionClient(address).GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.PlanID != "plus" || got.QuotaBytes != 100<<30 || got.UsedBytes != 2<<30 {
		t.Errorf("subscription = %+v, want the plan plus, the quota of 100 GiB and 2 GiB used", got)
	}
}

func TestGetSubscription_ReadsUsedBytesAboveTheQuota(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.state = &syncv1.GetSubscriptionResponse{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, Plan: "basic", QuotaBytes: 10 << 30, UsedBytes: 20 << 30,
	}

	got, err := newSubscriptionClient(address).GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.UsedBytes != 20<<30 || got.QuotaBytes != 10<<30 {
		t.Errorf("subscription = %+v, want 20 GiB used of 10 GiB without a clamp", got)
	}
}

func TestListPlans_ReturnsThePlansInTheOrderOfTheServer(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.plans = []*syncv1.Plan{
		{Id: "basic", Name: "Basic", QuotaBytes: 10 << 30, PriceCents: 500, Currency: "USD"},
		{Id: "plus", Name: "Plus", QuotaBytes: 100 << 30, PriceCents: 1500, Currency: "USD"},
		{Id: "pro", Name: "Pro", QuotaBytes: 250 << 30, PriceCents: 3000, Currency: "USD"},
	}

	got, err := newSubscriptionClient(address).ListPlans(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []syncprotocol.Plan{
		{ID: "basic", Name: "Basic", QuotaBytes: 10 << 30, PriceCents: 500, Currency: "USD"},
		{ID: "plus", Name: "Plus", QuotaBytes: 100 << 30, PriceCents: 1500, Currency: "USD"},
		{ID: "pro", Name: "Pro", QuotaBytes: 250 << 30, PriceCents: 3000, Currency: "USD"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("plans = %+v, want %+v", got, want)
	}
}

func TestListPlans_SendsTheVersionAndTheBearerOfTheToken(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)

	if _, err := newSubscriptionClient(address).ListPlans(t.Context()); err != nil {
		t.Fatal(err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 1 || stub.headers[0].Get("Authorization") != "Bearer token-of-the-session" {
		t.Errorf("headers = %v, want one request with the bearer of the token", stub.headers)
	}
}

func TestListPlans_ReportsAServerThatSellsNoSubscriptionsAsBillingUnavailable(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeUnimplemented, errors.New("no billing"))

	_, err := newSubscriptionClient(address).ListPlans(t.Context())

	if !errors.Is(err, syncprotocol.ErrBillingUnavailable) {
		t.Errorf("error = %v, want ErrBillingUnavailable", err)
	}
}

func TestCreateCheckout_SendsThePlanToTheServer(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.checkout = "https://pay.example/checkout/abc"

	if _, err := newSubscriptionClient(address).CreateCheckout(t.Context(), "plus"); err != nil {
		t.Fatal(err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if !slices.Equal(stub.asked, []string{"plus"}) {
		t.Errorf("plans asked = %v, want [plus]", stub.asked)
	}
}

func TestCreateCheckout_ReportsAnUnknownPlan(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeInvalidArgument, errors.New("no such plan"))

	_, err := newSubscriptionClient(address).CreateCheckout(t.Context(), "gold")

	if !errors.Is(err, syncprotocol.ErrUnknownPlan) {
		t.Errorf("error = %v, want ErrUnknownPlan", err)
	}
}

func TestChangePlan_SendsThePlanAndReturnsTheSubscriptionAfterTheChange(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.state = &syncv1.GetSubscriptionResponse{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, PeriodEndAt: 1_700_000_000, Plan: "pro", QuotaBytes: 250 << 30, UsedBytes: 1 << 30,
	}

	got, err := newSubscriptionClient(address).ChangePlan(t.Context(), "pro")
	if err != nil {
		t.Fatal(err)
	}

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if !slices.Equal(stub.asked, []string{"pro"}) {
		t.Errorf("plans asked = %v, want [pro]", stub.asked)
	}
	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE || got.PlanID != "pro" || got.QuotaBytes != 250<<30 || got.UsedBytes != 1<<30 {
		t.Errorf("subscription = %+v, want active on the plan pro with 250 GiB", got)
	}
	if want := time.Unix(1_700_000_000, 0); !got.PeriodEnd.Equal(want) {
		t.Errorf("PeriodEnd = %v, want %v", got.PeriodEnd, want)
	}
}

func TestChangePlan_MapsServerFailuresToTheErrorsOfThePackage(t *testing.T) {
	t.Parallel()
	for name, one := range map[string]struct {
		err  error
		want error
	}{
		"no active subscription": {err: connect.NewError(connect.CodeFailedPrecondition, errors.New("none")), want: syncprotocol.ErrNoActiveSubscription},
		"same plan":              {err: connect.NewError(connect.CodeAlreadyExists, errors.New("same")), want: syncprotocol.ErrSamePlan},
		"unknown plan":           {err: connect.NewError(connect.CodeInvalidArgument, errors.New("no")), want: syncprotocol.ErrUnknownPlan},
		"unimplemented":          {err: connect.NewError(connect.CodeUnimplemented, errors.New("no billing")), want: syncprotocol.ErrBillingUnavailable},
		"unauthenticated":        {err: connect.NewError(connect.CodeUnauthenticated, errors.New("no")), want: syncprotocol.ErrUnauthenticated},
		"unavailable":            {err: connect.NewError(connect.CodeUnavailable, errors.New("down")), want: syncprotocol.ErrUnreachable},
	} {
		stub, address := serveSubscriptionStub(t, syncprotocol.Version)
		stub.fail = one.err

		_, err := newSubscriptionClient(address).ChangePlan(t.Context(), "plus")

		if !errors.Is(err, one.want) {
			t.Errorf("ChangePlan of %s: error = %v, want %v", name, err, one.want)
		}
	}
}

func TestChangePlan_ReportsASamePlanNotAsAnActiveSubscription(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version)
	stub.fail = connect.NewError(connect.CodeAlreadyExists, errors.New("same"))

	_, err := newSubscriptionClient(address).ChangePlan(t.Context(), "plus")

	if errors.Is(err, syncprotocol.ErrAlreadySubscribed) {
		t.Errorf("error = %v, want ErrSamePlan and not ErrAlreadySubscribed", err)
	}
}

func TestChangePlan_ReportsAServerOfAnotherVersionAsUpdateRequired(t *testing.T) {
	t.Parallel()
	stub, address := serveSubscriptionStub(t, syncprotocol.Version+1)

	_, err := newSubscriptionClient(address).ChangePlan(t.Context(), "plus")

	if !errors.Is(err, syncprotocol.ErrUpdateRequired) || errors.Is(err, syncprotocol.ErrNoActiveSubscription) {
		t.Errorf("error = %v, want ErrUpdateRequired and not ErrNoActiveSubscription", err)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 0 {
		t.Errorf("the server ran %d calls, want none", len(stub.headers))
	}
}
