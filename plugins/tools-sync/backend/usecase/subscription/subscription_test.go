package subscription_test

import (
	"context"
	"errors"
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
	err      error
}

func (s server) GetSubscription(context.Context) (syncprotocol.Subscription, error) {
	return s.state, s.err
}

func (s server) CreateCheckout(context.Context) (string, error) { return s.checkout, s.err }

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
	got, err := subscription.NewCreateCheckout(subscription.Deps{Subscriptions: server{checkout: "https://pay.example/checkout/abc"}}).Execute(t.Context())

	if err != nil || got != "https://pay.example/checkout/abc" {
		t.Errorf("url = %q, error = %v, want the address of the server", got, err)
	}
}

func TestCreateCheckout_ReturnsAlreadySubscribedAndBillingUnavailableAsTheServerSentThem(t *testing.T) {
	for _, want := range []error{syncprotocol.ErrAlreadySubscribed, syncprotocol.ErrBillingUnavailable} {
		_, err := subscription.NewCreateCheckout(subscription.Deps{Subscriptions: server{err: want}}).Execute(t.Context())

		if !errors.Is(err, want) {
			t.Errorf("error = %v, want %v", err, want)
		}
	}
}
