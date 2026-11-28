package syncprotocol

import (
	"time"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// SubscriptionPhase is the phase of a subscription as the server tells it.
type SubscriptionPhase = syncv1.SubscriptionPhase

// Plan is a plan the server sells: its id, its name, how many bytes it holds and its price for one period in cents
// of the currency, tax included. The price is information; the payment page holds the one that is charged.
type Plan struct {
	ID         string
	Name       string
	QuotaBytes uint64
	PriceCents uint64
	Currency   string
}

// Subscription is the subscription of an account. A time the server did not send is the zero time. PlanID is empty
// and QuotaBytes is 0 while the phase is none. UsedBytes may be above QuotaBytes after a change to a smaller plan. IsTrial
// is true while a trial runs that no payment has followed; PeriodEnd is then the end of the trial.
type Subscription struct {
	Phase      SubscriptionPhase
	PeriodEnd  time.Time
	WritesEnd  time.Time
	DeleteAt   time.Time
	PlanID     string
	QuotaBytes uint64
	UsedBytes  uint64
	IsTrial    bool
}
