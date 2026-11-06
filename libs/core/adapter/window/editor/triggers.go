package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// runTriggers runs the triggers over the vault once, and again after each wake and each change of the
// settings, until ctx ends. A plugin turned on or off is a change of the settings. Wakes that arrive
// during a run make one run more.
func runTriggers(
	ctx context.Context, triggers *trigger.Triggers, v domain.Vault, wake <-chan struct{},
	watch *settings.Watch, report func(error),
) {
	var changes <-chan []settings.Setting
	if watch != nil {
		var stop func()
		changes, stop = watch.Execute()
		defer stop()
	}
	for {
		if err := triggers.Execute(ctx, v); err != nil && ctx.Err() == nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case _, ok := <-changes:
			if !ok {
				changes = nil
			}
		}
	}
}

// signal wakes what waits on the channel, and leaves one wake pending where one is already.
func signal(wake chan<- struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}
