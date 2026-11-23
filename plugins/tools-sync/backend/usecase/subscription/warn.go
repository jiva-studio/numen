package subscription

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Key is the key of the health item that tells the subscription does not allow changes, or will not for long.
const Key = "subscription-needed"

const (
	needsText  = "Sync needs a subscription"
	stayText   = "Subscription ended: changes stay on this device"
	graceHead  = "Subscription ended: changes are sent until "
	dateLayout = "2 Jan 2006"
)

// GetWarningText is what the health item says in the phase of the state. An active subscription has none.
func GetWarningText(state domain.State) string {
	switch state.Phase {
	case domain.PhaseActive:
		return ""
	case domain.PhaseGrace:
		return graceHead + state.WritesEnd.Format(dateLayout)
	case domain.PhaseReadOnly, domain.PhaseDeleted:
		return stayText
	case domain.PhaseNone:
	}
	return needsText
}

// WarnDeps are what Warn needs.
type WarnDeps struct {
	Subscriptions ports.Subscriptions
	Health        ports.Health
}

// Warn tells, once per round of sync, what the subscription means for the changes.
type Warn struct {
	read   *ReadSubscription
	health ports.Health
}

// NewWarn returns the use case over the dependencies.
func NewWarn(deps WarnDeps) *Warn {
	return &Warn{read: NewReadSubscription(Deps{Subscriptions: deps.Subscriptions}), health: deps.Health}
}

// Execute reads the subscription and raises the health item in every phase but active, which clears it. An
// error of the server is returned and leaves the item as it is. A refused write raises the same text.
func (u *Warn) Execute(ctx context.Context) error {
	state, err := u.read.Execute(ctx)
	if err != nil {
		return fmt.Errorf("read the subscription: %w", err)
	}
	text := GetWarningText(state)
	if text == "" {
		return u.health.Clear(ctx, Key)
	}
	return u.health.Report(ctx, health.Item{Key: Key, Severity: health.Warning, Text: text, Detail: "", Action: nil})
}
