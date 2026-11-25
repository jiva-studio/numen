package fakeserver_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const gib = 1 << 30

func (b *billing) payPlan(plan string) {
	b.tb.Helper()
	url, err := b.subscription.CreateCheckout(b.tb.Context(), plan)
	if err != nil {
		b.tb.Fatal(err)
	}
	if status, _ := b.get(url); status != http.StatusOK {
		b.tb.Fatalf("the pay page answered %d, want 200", status)
	}
}

func (b *billing) setUsage(bytes uint64) {
	b.tb.Helper()
	request, err := http.NewRequestWithContext(b.tb.Context(), http.MethodPost, b.address+"/admin/usage?bytes="+strconv.FormatUint(bytes, 10), nil)
	if err != nil {
		b.tb.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		b.tb.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		b.tb.Error(err)
	}
	if response.StatusCode >= 300 {
		b.tb.Fatalf("the usage admin path answered %d", response.StatusCode)
	}
}

func (b *billing) subscriptionNow() syncprotocol.Subscription {
	b.tb.Helper()
	got, err := b.subscription.GetSubscription(b.tb.Context())
	if err != nil {
		b.tb.Fatal(err)
	}
	return got
}

func (b *billing) reserve(vault []byte, size uint64) error {
	b.tb.Helper()
	_, err := b.env.blob.ReserveBlobs(b.tb.Context(), vault, []uint64{size})
	return err
}

func TestListPlans_ReturnsBasicPlusAndProByDefault(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	got, err := b.subscription.ListPlans(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []syncprotocol.Plan{
		{ID: "basic", Name: "Basic", QuotaBytes: 10 * gib, PriceCents: 500, Currency: "USD"},
		{ID: "plus", Name: "Plus", QuotaBytes: 100 * gib, PriceCents: 1500, Currency: "USD"},
		{ID: "pro", Name: "Pro", QuotaBytes: 250 * gib, PriceCents: 3000, Currency: "USD"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("plans = %+v, want %+v", got, want)
	}
}

func TestCreateCheckout_RefusesAnEmptyOrUnknownPlan(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	for _, plan := range []string{"", "gold"} {
		_, err := b.subscription.CreateCheckout(t.Context(), plan)

		if !errors.Is(err, syncprotocol.ErrUnknownPlan) {
			t.Errorf("plan %q: error = %v, want ErrUnknownPlan", plan, err)
		}
	}
}

func TestGetSubscription_ReportsThePlanAndTheQuotaOfThePlanThatWasPaid(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	b.payPlan("plus")

	got := b.subscriptionNow()
	if got.PlanID != "plus" || got.QuotaBytes != 100*gib || got.UsedBytes != 0 {
		t.Errorf("subscription = %+v, want the plan plus with 100 GiB and nothing used", got)
	}
}

func TestGetSubscription_HasNoPlanBeforeAPayment(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	got := b.subscriptionNow()

	if got.PlanID != "" || got.QuotaBytes != 0 || got.UsedBytes != 0 {
		t.Errorf("subscription = %+v, want no plan, no quota and nothing used", got)
	}
}

func TestGetSubscription_KeepsThePlanAfterTheSubscriptionEnded(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("plus")
	b.setPhase("phase=read_only")

	got := b.subscriptionNow()

	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY || got.PlanID != "plus" || got.QuotaBytes != 100*gib {
		t.Errorf("subscription = %+v, want read only on the plan plus", got)
	}
}

func TestGetSubscription_ReportsTheUsedBytes(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("plus")

	b.setUsage(2 * gib)

	if got := b.subscriptionNow(); got.UsedBytes != 2*gib {
		t.Errorf("UsedBytes = %d, want 2 GiB", got.UsedBytes)
	}
}

func TestChangePlan_MovesAnActiveSubscriptionToTheLargerPlan(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("basic")

	got, err := b.subscription.ChangePlan(t.Context(), "pro")
	if err != nil {
		t.Fatal(err)
	}

	if got.PlanID != "pro" || got.QuotaBytes != 250*gib || got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		t.Errorf("subscription = %+v, want active on the plan pro with 250 GiB", got)
	}
	if again := b.subscriptionNow(); again.PlanID != "pro" || again.QuotaBytes != 250*gib {
		t.Errorf("GetSubscription = %+v, want the plan pro", again)
	}
}

func TestChangePlan_MovesAnActiveSubscriptionToTheSmallerPlan(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("pro")

	got, err := b.subscription.ChangePlan(t.Context(), "plus")
	if err != nil {
		t.Fatal(err)
	}

	if got.PlanID != "plus" || got.QuotaBytes != 100*gib {
		t.Errorf("subscription = %+v, want the plan plus with 100 GiB", got)
	}
}

func TestChangePlan_RefusesThePlanTheAccountHas(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("plus")

	_, err := b.subscription.ChangePlan(t.Context(), "plus")

	if !errors.Is(err, syncprotocol.ErrSamePlan) {
		t.Errorf("error = %v, want ErrSamePlan", err)
	}
}

func TestChangePlan_RefusesAnUnknownPlan(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("plus")

	_, err := b.subscription.ChangePlan(t.Context(), "gold")

	if !errors.Is(err, syncprotocol.ErrUnknownPlan) {
		t.Errorf("error = %v, want ErrUnknownPlan", err)
	}
}

func TestChangePlan_RefusesAnAccountWithoutAnActiveSubscription(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"none", "grace", "read_only", "deleted"} {
		b := createBillingEnv(t, true)
		b.payPlan("plus")
		b.setPhase("phase=" + phase)

		_, err := b.subscription.ChangePlan(t.Context(), "pro")

		if !errors.Is(err, syncprotocol.ErrNoActiveSubscription) {
			t.Errorf("phase %s: error = %v, want ErrNoActiveSubscription", phase, err)
		}
	}
}

func TestReserveBlobs_RefusesBytesAbovePlanQuotaWithTheLimitOfThePlan(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("basic")
	vault := b.env.newVault()
	b.setUsage(10 * gib)

	err := b.reserve(vault, 1)

	var exceeded syncprotocol.QuotaExceededError
	if !errors.As(err, &exceeded) || exceeded.Limit != 10*gib || exceeded.Used != 10*gib {
		t.Errorf("error = %v, want QuotaExceededError with 10 GiB used of 10 GiB", err)
	}
}

func TestReserveBlobs_AcceptsBytesAfterThePlanBecameLarger(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("basic")
	vault := b.env.newVault()
	b.setUsage(10 * gib)
	if _, err := b.subscription.ChangePlan(t.Context(), "plus"); err != nil {
		t.Fatal(err)
	}

	if err := b.reserve(vault, 1); err != nil {
		t.Errorf("a reservation after the change to a larger plan failed: %v", err)
	}
}

func TestReserveBlobs_RefusesNewBytesWhenThePlanBecameSmallerThanTheUsedBytes(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("pro")
	vault := b.env.newVault()
	b.setUsage(20 * gib)
	if _, err := b.subscription.ChangePlan(t.Context(), "basic"); err != nil {
		t.Fatal(err)
	}

	err := b.reserve(vault, 1)

	var exceeded syncprotocol.QuotaExceededError
	if !errors.As(err, &exceeded) || exceeded.Used != 20*gib || exceeded.Limit != 10*gib {
		t.Errorf("error = %v, want QuotaExceededError with 20 GiB used of 10 GiB", err)
	}
	if got := b.subscriptionNow(); got.UsedBytes != 20*gib || got.QuotaBytes != 10*gib {
		t.Errorf("subscription = %+v, want 20 GiB used of 10 GiB", got)
	}
}

func TestListPlans_ReturnsThePlansOfTheConfigOfTheServer(t *testing.T) {
	t.Parallel()
	want := []syncprotocol.Plan{
		{ID: "tiny", Name: "Tiny", QuotaBytes: 1024, PriceCents: 99, Currency: "EUR"},
		{ID: "huge", Name: "Huge", QuotaBytes: 1 << 20, PriceCents: 4900, Currency: "EUR"},
	}
	_, plans := newPlansServer(t, want)

	got, err := plans.ListPlans(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(got, want) {
		t.Errorf("plans = %+v, want %+v", got, want)
	}
}

func TestChangePlan_RefusesACallWithoutASession(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.payPlan("plus")
	anonymous := newAnonymousSubscription(b.address)

	_, err := anonymous.ChangePlan(t.Context(), "pro")

	if !isUnauthenticated(err) {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}

func newPlansServer(tb testing.TB, plans []syncprotocol.Plan) (*billing, *syncclient.SubscriptionClient) {
	tb.Helper()
	clock := &testClock{now: machineClock{}.Now()}
	server := fakeserver.New(fakeserver.Config{Now: clock.Now, SessionTTL: 1000 * day, NeedsSubscription: true, Plans: plans})
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "First")
	b := newBillingOf(tb, server, clock, served.URL, &account, device)
	return b, b.subscription
}

func newAnonymousSubscription(address string) *syncclient.SubscriptionClient {
	return syncclient.NewSubscriptionClient(address, http.DefaultClient, func() string { return "" })
}

func TestAdminTrial_AnswersTheTimeTheTrialEndsAndStoresIt(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, b.address+"/admin/trial?plan=plus", nil)
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
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	want := b.clock.now.Add(14 * day).UTC().Format(time.RFC3339)
	if got := strings.TrimSpace(string(body)); response.StatusCode != http.StatusOK || got != want {
		t.Errorf("answer = %d %q, want 200 %q", response.StatusCode, got, want)
	}
	if got := b.subscriptionNow(); got.PlanID != "plus" {
		t.Errorf("subscription = %+v, want the plan plus", got)
	}
}
