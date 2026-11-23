// Package subscription is the subscription of the account as the sync server tells it. The server decides the
// phase and the times; nothing here computes a date.
package subscription

import "time"

// Phase is where the subscription stands.
type Phase int

const (
	// PhaseNone is an account that never had a subscription.
	PhaseNone Phase = iota + 1
	// PhaseActive is a paid period that runs.
	PhaseActive
	// PhaseGrace is a period that ended and whose changes are still sent until WritesEnd.
	PhaseGrace
	// PhaseReadOnly is an account whose changes the server refuses and whose vaults it deletes at DeleteAt.
	PhaseReadOnly
	// PhaseDeleted is an account whose vaults the server deleted.
	PhaseDeleted
)

// State is the subscription with its times. A time the server did not send is the zero time.
type State struct {
	Phase     Phase
	PeriodEnd time.Time
	WritesEnd time.Time
	DeleteAt  time.Time
	// PlanID is empty while the phase is none. UsedBytes may be above QuotaBytes.
	PlanID     string
	QuotaBytes uint64
	UsedBytes  uint64
	// IsTrial is true while a trial runs that no payment has followed; PeriodEnd is then the end of the trial.
	IsTrial bool
}

// CanWrite is whether the server takes changes.
func (s State) CanWrite() bool {
	return s.Phase == PhaseActive || s.Phase == PhaseGrace
}

// Plan is a plan the server sells.
type Plan struct {
	ID         string
	Name       string
	QuotaBytes uint64
	PriceCents uint64
	Currency   string
}
