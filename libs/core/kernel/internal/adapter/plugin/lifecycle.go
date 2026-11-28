package plugin

import (
	"context"
	"time"
)

type pluginTeardown interface {
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
}

type pluginKeeper interface {
	KeepAll(ctx context.Context)
}

// Lifecycle stops the plugins when the host shuts down and starts the ones that stay running.
type Lifecycle struct {
	teardown pluginTeardown
	keeper   pluginKeeper
}

func NewLifecycle(teardown pluginTeardown, keeper pluginKeeper) Lifecycle {
	return Lifecycle{teardown: teardown, keeper: keeper}
}

// TeardownAll stops every plugin within the grace period.
func (l Lifecycle) TeardownAll(ctx context.Context, gracePeriod time.Duration) error {
	return l.teardown.TeardownAll(ctx, gracePeriod)
}

// KeepAll starts every enabled plugin that stays running.
func (l Lifecycle) KeepAll(ctx context.Context) { l.keeper.KeepAll(ctx) }
