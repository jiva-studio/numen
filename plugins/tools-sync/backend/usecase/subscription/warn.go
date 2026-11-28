package subscription

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

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
	text := domain.GetWarningText(state)
	if text == "" {
		return u.health.Clear(ctx, domain.WarningKey)
	}
	return u.health.Report(ctx, health.Item{Key: domain.WarningKey, Severity: health.Warning, Text: text, Detail: "", Action: nil})
}
