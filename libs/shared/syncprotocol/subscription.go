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
// and QuotaBytes is 0 while the phase is none. UsedBytes may be above QuotaBytes after a change to a smaller plan.
type Subscription struct {
	Phase      SubscriptionPhase
	PeriodEnd  time.Time
	WritesEnd  time.Time
	DeleteAt   time.Time
	PlanID     string
	QuotaBytes uint64
	UsedBytes  uint64
	// IsTrial is true while a trial runs that no payment has followed; PeriodEnd is then the end of the trial.
	IsTrial bool
}

// Product is a product the server sells that is no plan: it has no quota. ID is what CreateCheckout takes. OneTime is true for a single payment
// and false for a subscription. The price is information; the payment page holds the one that is charged.
type Product struct {
	ID         string
	Slug       string
	Name       string
	PriceCents uint64
	Currency   string
	OneTime    bool
}

// Entitlement is one product the account owns right now. Phase is ACTIVE or GRACE. ValidUntil is the end of the paid period or of the trial, and about
// 100 years after the purchase for a one-time product. QuotaBytes is 0 for a product without a quota.
type Entitlement struct {
	Product    string
	ValidUntil time.Time
	Phase      SubscriptionPhase
	QuotaBytes uint64
	IsTrial    bool
}
