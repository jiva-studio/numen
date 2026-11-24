package plugin

import (
	"context"
	"time"
)

// Lifecycle stops the plugins when the host shuts down and starts the ones that stay running.
type Lifecycle struct {
	teardown *Teardown
	keeper   *Keeper
}

// NewLifecycle creates a lifecycle from the teardown and the keeper of the plugins.
func NewLifecycle(teardown *Teardown, keeper *Keeper) Lifecycle {
	return Lifecycle{teardown: teardown, keeper: keeper}
}

// TeardownAll stops every plugin within the grace period.
func (l Lifecycle) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	return l.teardown.TeardownAll(ctx, gracePeriod)
}

// KeepAll starts every enabled plugin that stays running.
func (l Lifecycle) KeepAll(ctx context.Context) { l.keeper.KeepAll(ctx) }
