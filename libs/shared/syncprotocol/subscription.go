package syncprotocol

import (
	"time"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// SubscriptionPhase is the phase of a subscription as the server tells it.
type SubscriptionPhase = syncv1.SubscriptionPhase

// Plan is a plan the server sells: its id and how many bytes it holds.
type Plan struct {
	ID         string
	QuotaBytes uint64
}

// Subscription is the subscription of an account. A time the server did not send is the zero time. PlanID is empty
// and QuotaBytes is 0 while the phase is none. UsedBytes may be above QuotaBytes after a change to a smaller plan.
type Subscription struct {
	Phase      SubscriptionPhase
	PeriodEnd  time.Time
	WritesEnd  time.Time
	DeleteAt   time.Time
	PlanID     string
	QuotaBytes uint64
	UsedBytes  uint64
}
