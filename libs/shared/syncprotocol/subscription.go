package syncprotocol

import (
	"time"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// SubscriptionPhase is the phase of a subscription as the server tells it.
type SubscriptionPhase = syncv1.SubscriptionPhase

// Subscription is the subscription of an account. A time the server did not send is the zero time.
type Subscription struct {
	Phase     SubscriptionPhase
	PeriodEnd time.Time
	WritesEnd time.Time
	DeleteAt  time.Time
}
