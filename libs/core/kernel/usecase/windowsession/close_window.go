package windowsession

import (
	"context"
	"errors"
	"time"
)

// ShutdownBound is how long a resource of the window may take to close.
const ShutdownBound = time.Second

// CloseWindow shuts the door on every question, stops the passes behind the vault, waits for them,
// and closes what the window holds open.
type CloseWindow struct {
	life *lifecycle
}

// Execute closes the window. The resources close in the order they are given, which is the order they
// depend on each other in.
func (u CloseWindow) Execute(ctx context.Context) error {
	deps := u.life.deps
	// Searches, note reads and link lookups read the index directly; the index
	// closes after the last of them has finished.
	deps.Shutdown.Door.Close()
	u.life.stopVault()

	bound, stop := context.WithTimeout(context.WithoutCancel(ctx), ShutdownBound)
	defer stop()
	var failures []error
	for _, closer := range deps.Shutdown.Closers {
		failures = append(failures, closer.Close(bound))
	}
	return errors.Join(failures...)
}
