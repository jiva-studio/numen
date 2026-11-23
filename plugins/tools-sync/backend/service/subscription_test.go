package service_test

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
)

func newBillingWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{NeedsSubscription: true})
	words := synctest.NewWords(t)
	return &world{server: server, clock: clock, words: words, c: synctest.NewFirstComputer(t, server, clock, "Test", words)}
}

func setPhase(t *testing.T, server *synctest.Server, phase string) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"/admin/subscription?phase="+phase, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.StatusCode >= 300 {
		t.Fatalf("the admin path answered %d", response.StatusCode)
	}
}

func pay(t *testing.T, url string) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("the pay page answered %d, want 200", response.StatusCode)
	}
}

func rfc3339(at time.Time) string { return at.UTC().Format(time.RFC3339) }

func TestGetSubscription_ReportsNoPhaseAndAsksNoServerWhenThisDeviceHasNoIdentity(t *testing.T) {
	w := newBillingWorld(t)
	w.c.Fake.NoIdentity()
	backend := service.NewBackend(w.c.SyncDeps())

	got, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if got != (api.GetSubscriptionResponse{}) {
		t.Errorf("response = %+v, want no identity, no phase and no times", got)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure); calls != 0 {
		t.Errorf("the server ran %d subscription calls, want none", calls)
	}
}

func TestGetSubscription_ReportsNoSubscriptionForANewAccount(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	got, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if want := (api.GetSubscriptionResponse{HasIdentity: true, Phase: api.PhaseNone}); got != want {
		t.Errorf("response = %+v, want %+v", got, want)
	}
}

func TestGetSubscription_ReportsTheActivePhaseWithTheTimesAsRFC3339(t *testing.T) {
	w := newWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	got, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}

	periodEnd := w.clock.Now().Add(30 * 24 * time.Hour)
	want := api.GetSubscriptionResponse{
		HasIdentity: true, Phase: api.PhaseActive, PeriodEndAt: rfc3339(periodEnd),
		WritesEndAt: rfc3339(periodEnd.Add(7 * 24 * time.Hour)), DeleteAt: rfc3339(periodEnd.Add(60 * 24 * time.Hour)),
		QuotaBytes: 10 << 30,
	}
	if got != want {
		t.Errorf("response = %+v, want %+v", got, want)
	}
}

func TestGetSubscription_ReportsEachPhaseTheServerSets(t *testing.T) {
	for name, one := range map[string]struct {
		admin string
		want  api.Phase
	}{
		"grace":     {admin: "grace", want: api.PhaseGrace},
		"read only": {admin: "read_only", want: api.PhaseReadOnly},
		"deleted":   {admin: "deleted", want: api.PhaseDeleted},
		"none":      {admin: "none", want: api.PhaseNone},
	} {
		w := newBillingWorld(t)
		setPhase(t, w.server, one.admin)
		backend := service.NewBackend(w.c.SyncDeps())

		got, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
		if err != nil {
			t.Fatal(err)
		}

		if got.Phase != one.want || !got.HasIdentity {
			t.Errorf("%s: response = %+v, want phase %s", name, got, one.want)
		}
		if one.want != api.PhaseNone && (got.PeriodEndAt == "" || got.WritesEndAt == "" || got.DeleteAt == "") {
			t.Errorf("%s: response = %+v, want the three times", name, got)
		}
	}
}

func TestGetSubscription_ReturnsTheErrorOfAServerThatIsAway(t *testing.T) {
	w := newBillingWorld(t)
	w.server.SetDown(true)
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})

	if !errors.Is(err, syncprotocol.ErrUnreachable) && !errors.Is(err, sessiondomain.ErrUnreachable) {
		t.Errorf("error = %v, want an unreachable server", err)
	}
}

func TestCreateCheckout_ReturnsThePaymentPageAndTheSubscriptionIsActiveAfterPayment(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	got, err := backend.CreateCheckout(t.Context(), api.CreateCheckoutRequest{Plan: "plus"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(got.CheckoutURL, w.server.URL+"/pay/") {
		t.Fatalf("checkout url = %q, want a page of the server", got.CheckoutURL)
	}
	pay(t, got.CheckoutURL)
	state, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != api.PhaseActive {
		t.Errorf("phase = %q, want active", state.Phase)
	}
}

func TestCreateCheckout_ReturnsAlreadySubscribedWhileTheSubscriptionIsActive(t *testing.T) {
	w := newBillingWorld(t)
	setPhase(t, w.server, "active")
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.CreateCheckout(t.Context(), api.CreateCheckoutRequest{Plan: "plus"})

	if !errors.Is(err, syncprotocol.ErrAlreadySubscribed) {
		t.Errorf("error = %v, want ErrAlreadySubscribed", err)
	}
}

func TestCreateCheckout_ReturnsNoIdentityAndAsksNoServerWhenThisDeviceHasNoIdentity(t *testing.T) {
	w := newBillingWorld(t)
	w.c.Fake.NoIdentity()
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.CreateCheckout(t.Context(), api.CreateCheckoutRequest{Plan: "plus"})

	if !errors.Is(err, sessiondomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.SubscriptionServiceCreateCheckoutProcedure); calls != 0 {
		t.Errorf("the server ran %d checkout calls, want none", calls)
	}
}

func TestGetStatus_ReportsNoSubscriptionItem(t *testing.T) {
	w := newBillingWorld(t)
	setPhase(t, w.server, "read_only")
	backend := service.NewBackend(w.c.SyncDeps())

	status, err := backend.GetStatus(t.Context(), api.GetStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if len(status.Items) != 0 {
		t.Errorf("items = %+v, want none", status.Items)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.SubscriptionServiceGetSubscriptionProcedure); calls != 0 {
		t.Errorf("GetStatus ran %d subscription calls, want none", calls)
	}
}

const gib = 1 << 30

func payFor(t *testing.T, backend *service.Backend, plan string) {
	t.Helper()
	got, err := backend.CreateCheckout(t.Context(), api.CreateCheckoutRequest{Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	pay(t, got.CheckoutURL)
}

func TestListPlans_ReturnsThePlansOfTheServerAscendingByQuota(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	got, err := backend.ListPlans(t.Context(), api.ListPlansRequest{})
	if err != nil {
		t.Fatal(err)
	}

	want := []api.Plan{{ID: "basic", QuotaBytes: 10 * gib}, {ID: "plus", QuotaBytes: 100 * gib}, {ID: "pro", QuotaBytes: 250 * gib}}
	if !slices.Equal(got.Plans, want) {
		t.Errorf("plans = %+v, want %+v", got.Plans, want)
	}
}

func TestListPlans_ReturnsNoIdentityAndAsksNoServerWhenThisDeviceHasNoIdentity(t *testing.T) {
	w := newBillingWorld(t)
	w.c.Fake.NoIdentity()
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.ListPlans(t.Context(), api.ListPlansRequest{})

	if !errors.Is(err, sessiondomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.SubscriptionServiceListPlansProcedure); calls != 0 {
		t.Errorf("the server ran %d plan calls, want none", calls)
	}
}

func TestGetSubscription_ReportsThePlanAfterThePersonPaidForIt(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	payFor(t, backend, "plus")

	got, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Plan != "plus" || got.QuotaBytes != 100*gib || got.UsedBytes != 0 {
		t.Errorf("response = %+v, want the plan plus with 100 GiB and nothing used", got)
	}
}

func TestGetSubscription_ReportsTheBytesTheCloudHolds(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())
	payFor(t, backend, "plus")
	w.server.SetUsedBytes(t, 2*gib)

	got, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}

	if got.UsedBytes != 2*gib || got.QuotaBytes != 100*gib {
		t.Errorf("response = %+v, want 2 GiB of 100 GiB used", got)
	}
}

func TestCreateCheckout_RefusesAnUnknownPlan(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.CreateCheckout(t.Context(), api.CreateCheckoutRequest{Plan: "gold"})

	if !errors.Is(err, syncprotocol.ErrUnknownPlan) {
		t.Errorf("error = %v, want ErrUnknownPlan", err)
	}
}

func TestChangePlan_ReturnsTheSubscriptionOnTheNewPlan(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())
	payFor(t, backend, "basic")

	got, err := backend.ChangePlan(t.Context(), api.ChangePlanRequest{Plan: "pro"})
	if err != nil {
		t.Fatal(err)
	}

	if got.Subscription.Plan != "pro" || got.Subscription.QuotaBytes != 250*gib || got.Subscription.Phase != api.PhaseActive || !got.Subscription.HasIdentity {
		t.Errorf("subscription = %+v, want active on the plan pro with 250 GiB", got.Subscription)
	}
	state, err := backend.GetSubscription(t.Context(), api.GetSubscriptionRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if state.Plan != "pro" {
		t.Errorf("GetSubscription plan = %q, want pro", state.Plan)
	}
}

func TestChangePlan_RefusesThePlanTheAccountHas(t *testing.T) {
	w := newBillingWorld(t)
	backend := service.NewBackend(w.c.SyncDeps())
	payFor(t, backend, "plus")

	_, err := backend.ChangePlan(t.Context(), api.ChangePlanRequest{Plan: "plus"})

	if !errors.Is(err, syncprotocol.ErrSamePlan) {
		t.Errorf("error = %v, want ErrSamePlan", err)
	}
}

func TestChangePlan_RefusesAnAccountWithoutAnActiveSubscription(t *testing.T) {
	for _, phase := range []string{"none", "grace", "read_only"} {
		w := newBillingWorld(t)
		backend := service.NewBackend(w.c.SyncDeps())
		payFor(t, backend, "plus")
		setPhase(t, w.server, phase)

		_, err := backend.ChangePlan(t.Context(), api.ChangePlanRequest{Plan: "pro"})

		if !errors.Is(err, syncprotocol.ErrNoActiveSubscription) {
			t.Errorf("phase %s: error = %v, want ErrNoActiveSubscription", phase, err)
		}
	}
}

func TestChangePlan_ReturnsNoIdentityAndAsksNoServerWhenThisDeviceHasNoIdentity(t *testing.T) {
	w := newBillingWorld(t)
	w.c.Fake.NoIdentity()
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.ChangePlan(t.Context(), api.ChangePlanRequest{Plan: "plus"})

	if !errors.Is(err, sessiondomain.ErrNoIdentity) {
		t.Errorf("error = %v, want ErrNoIdentity", err)
	}
	if calls := w.server.Fake.CountCalls(syncv1connect.SubscriptionServiceChangePlanProcedure); calls != 0 {
		t.Errorf("the server ran %d change calls, want none", calls)
	}
}
