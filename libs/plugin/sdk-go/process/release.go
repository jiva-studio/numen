package process

import (
	"context"
	"log/slog"
)

// Releaser tells the host that the process of the plugin has finished its work. A Client is one.
type Releaser interface {
	Release(ctx context.Context) error
}

// ReleaseAndLog releases the process through releaser and logs a failure; a process the host did not
// release stops at its idle limit. A nil releaser releases nothing.
func ReleaseAndLog(ctx context.Context, releaser Releaser) {
	if releaser == nil {
		return
	}
	if err := releaser.Release(ctx); err != nil {
		slog.Warn("release the process", "error", err)
	}
}
