package synctest

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type phaseSource struct {
	ports.Subscriptions

	subscription syncprotocol.Subscription
}

func (p phaseSource) GetSubscription(context.Context) (syncprotocol.Subscription, error) {
	return p.subscription, nil
}

func (c *Computer) getSubscriptions() ports.Subscriptions {
	if c.subscriptions == nil {
		return c.Cloud
	}
	return c.subscriptions
}

// ShowPhase makes the plugin of this computer read the phase from a source of the test, while the server keeps the subscription of the account as it is: the
// server clock cannot move the account past its grace, so the way the plugin treats the phase is tested over its port. The phase is "none", "active", "grace",
// "read_only" or "deleted".
func (c *Computer) ShowPhase(t *testing.T, phase string) {
	t.Helper()
	now := c.Clock.Now()
	subscription := syncprotocol.Subscription{
		Phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE, PeriodEnd: time.Time{}, WritesEnd: time.Time{}, DeleteAt: time.Time{},
		PlanID: "", QuotaBytes: 0, UsedBytes: 0, IsTrial: false,
	}
	ended := func(phase syncv1.SubscriptionPhase, ago time.Duration) syncprotocol.Subscription {
		end := now.Add(-ago)
		return syncprotocol.Subscription{
			Phase: phase, PeriodEnd: end, WritesEnd: end.Add(7 * day), DeleteAt: end.Add(60 * day), PlanID: DefaultPlan, QuotaBytes: 10 << 30, UsedBytes: 0, IsTrial: false,
		}
	}
	switch phase {
	case "none":
	case "active":
		subscription = ended(syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, -subscribePeriod)
	case "grace":
		subscription = ended(syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, day)
	case "read_only":
		subscription = ended(syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, 8*day)
	case "deleted":
		subscription = ended(syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED, 61*day)
	default:
		t.Fatalf("unknown phase %q", phase)
	}
	c.subscriptions = phaseSource{Subscriptions: c.Cloud, subscription: subscription}
}
