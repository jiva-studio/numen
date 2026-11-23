package subscription_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/subscription"
)

type server struct {
	state    syncprotocol.Subscription
	checkout string
	plans    []syncprotocol.Plan
	err      error
}

func (s server) GetSubscription(context.Context) (syncprotocol.Subscription, error) {
	return s.state, s.err
}

func (s server) CreateCheckout(context.Context, string) (string, error) { return s.checkout, s.err }

func (s server) ListPlans(context.Context) ([]syncprotocol.Plan, error) { return s.plans, s.err }

func (s server) ChangePlan(context.Context, string) (syncprotocol.Subscription, error) {
	return s.state, s.err
}

func TestReadSubscription_ReturnsThePhaseAndTheTimesTheServerSent(t *testing.T) {
	periodEnd := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	for name, one := range map[string]struct {
		wire syncv1.SubscriptionPhase
		want domain.Phase
	}{
		"none":      {wire: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE, want: domain.PhaseNone},
		"active":    {wire: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, want: domain.PhaseActive},
		"grace":     {wire: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, want: domain.PhaseGrace},
		"read only": {wire: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, want: domain.PhaseReadOnly},
		"deleted":   {wire: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED, want: domain.PhaseDeleted},
	} {
		state := syncprotocol.Subscription{Phase: one.wire, PeriodEnd: periodEnd, WritesEnd: periodEnd.Add(7 * 24 * time.Hour), DeleteAt: periodEnd.Add(60 * 24 * time.Hour)}

		got, err := subscription.NewReadSubscription(subscription.Deps{Subscriptions: server{state: state}}).Execute(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if got.Phase != one.want || !got.PeriodEnd.Equal(state.PeriodEnd) || !got.WritesEnd.Equal(state.WritesEnd) || !got.DeleteAt.Equal(state.DeleteAt) {
			t.Errorf("%s: state = %+v, want phase %v with the times of the server", name, got, one.want)
		}
	}
}

func TestReadSubscription_RefusesAPhaseItDoesNotKnow(t *testing.T) {
	state := syncprotocol.Subscription{Phase: syncv1.SubscriptionPhase(42)}

	_, err := subscription.NewReadSubscription(subscription.Deps{Subscriptions: server{state: state}}).Execute(t.Context())

	if err == nil {
		t.Error("Execute of an unknown phase succeeded, want an error")
	}
}

func TestReadSubscription_ReturnsTheErrorOfTheServer(t *testing.T) {
	_, err := subscription.NewReadSubscription(subscription.Deps{Subscriptions: server{err: syncprotocol.ErrUnreachable}}).Execute(t.Context())

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
}

func TestCreateCheckout_ReturnsTheAddressOfThePaymentPage(t *testing.T) {
	got, err := subscription.NewCreateCheckout(subscription.Deps{Subscriptions: server{checkout: "https://pay.example/checkout/abc"}}).Execute(t.Context(), "plus")

	if err != nil || got != "https://pay.example/checkout/abc" {
		t.Errorf("url = %q, error = %v, want the address of the server", got, err)
	}
}

func TestCreateCheckout_ReturnsAlreadySubscribedAndBillingUnavailableAsTheServerSentThem(t *testing.T) {
	for _, want := range []error{syncprotocol.ErrAlreadySubscribed, syncprotocol.ErrBillingUnavailable} {
		_, err := subscription.NewCreateCheckout(subscription.Deps{Subscriptions: server{err: want}}).Execute(t.Context(), "plus")

		if !errors.Is(err, want) {
			t.Errorf("error = %v, want %v", err, want)
		}
	}
}

func TestReadSubscription_ReturnsThePlanTheQuotaAndTheUsedBytes(t *testing.T) {
	state := syncprotocol.Subscription{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, PlanID: "plus", QuotaBytes: 100 << 30, UsedBytes: 2 << 30,
	}

	got, err := subscription.NewReadSubscription(subscription.Deps{Subscriptions: server{state: state}}).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.PlanID != "plus" || got.QuotaBytes != 100<<30 || got.UsedBytes != 2<<30 {
		t.Errorf("state = %+v, want the plan plus with 2 GiB of 100 GiB used", got)
	}
}

func TestReadSubscription_KeepsTheUsedBytesAboveTheQuota(t *testing.T) {
	state := syncprotocol.Subscription{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, PlanID: "basic", QuotaBytes: 10 << 30, UsedBytes: 20 << 30,
	}

	got, err := subscription.NewReadSubscription(subscription.Deps{Subscriptions: server{state: state}}).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.UsedBytes != 20<<30 || got.QuotaBytes != 10<<30 {
		t.Errorf("state = %+v, want 20 GiB used of 10 GiB without a clamp", got)
	}
}

func TestReadSubscription_KeepsThePlanOfAnEndedSubscription(t *testing.T) {
	state := syncprotocol.Subscription{Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, PlanID: "plus", QuotaBytes: 100 << 30}

	got, err := subscription.NewReadSubscription(subscription.Deps{Subscriptions: server{state: state}}).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Phase != domain.PhaseReadOnly || got.PlanID != "plus" || got.QuotaBytes != 100<<30 {
		t.Errorf("state = %+v, want read only on the plan plus", got)
	}
}

func TestListPlans_ReturnsThePlansInTheOrderOfTheServer(t *testing.T) {
	plans := []syncprotocol.Plan{
		{ID: "basic", Name: "Basic", QuotaBytes: 10 << 30, PriceCents: 500, Currency: "USD"},
		{ID: "plus", Name: "Plus", QuotaBytes: 100 << 30, PriceCents: 1500, Currency: "USD"},
	}

	got, err := subscription.NewListPlans(subscription.Deps{Subscriptions: server{plans: plans}}).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.Plan{
		{ID: "basic", Name: "Basic", QuotaBytes: 10 << 30, PriceCents: 500, Currency: "USD"},
		{ID: "plus", Name: "Plus", QuotaBytes: 100 << 30, PriceCents: 1500, Currency: "USD"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("plans = %+v, want %+v", got, want)
	}
}

func TestListPlans_ReturnsTheErrorOfTheServer(t *testing.T) {
	_, err := subscription.NewListPlans(subscription.Deps{Subscriptions: server{err: syncprotocol.ErrBillingUnavailable}}).Execute(t.Context())

	if !errors.Is(err, syncprotocol.ErrBillingUnavailable) {
		t.Errorf("error = %v, want ErrBillingUnavailable", err)
	}
}

func TestChangePlan_ReturnsTheSubscriptionAfterTheChange(t *testing.T) {
	state := syncprotocol.Subscription{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, PlanID: "pro", QuotaBytes: 250 << 30, UsedBytes: 1 << 30,
	}

	got, err := subscription.NewChangePlan(subscription.Deps{Subscriptions: server{state: state}}).Execute(t.Context(), "pro")
	if err != nil {
		t.Fatal(err)
	}

	if got.Phase != domain.PhaseActive || got.PlanID != "pro" || got.QuotaBytes != 250<<30 || got.UsedBytes != 1<<30 {
		t.Errorf("state = %+v, want active on the plan pro with 250 GiB", got)
	}
}

func TestChangePlan_ReturnsServerErrorsUnchanged(t *testing.T) {
	for _, want := range []error{
		syncprotocol.ErrNoActiveSubscription, syncprotocol.ErrSamePlan, syncprotocol.ErrUnknownPlan, syncprotocol.ErrBillingUnavailable, syncprotocol.ErrUnreachable,
	} {
		_, err := subscription.NewChangePlan(subscription.Deps{Subscriptions: server{err: want}}).Execute(t.Context(), "plus")

		if !errors.Is(err, want) {
			t.Errorf("error = %v, want %v", err, want)
		}
	}
}
