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
	door    Door
	closers []Closer
	session vaultSession
}

// NewCloseWindow makes the use case over the door, the session of the window and the resources it
// closes, in the order they depend on each other in.
func NewCloseWindow(door Door, session *Session, closers ...Closer) CloseWindow {
	return CloseWindow{door: door, closers: closers, session: session}
}

// Execute closes the window. The resources close in the order they are given, which is the order they
// depend on each other in.
func (u CloseWindow) Execute(ctx context.Context) error {
	// Searches, note reads and link lookups read the index directly; the index
	// closes after the last of them has finished.
	u.door.Close()
	u.session.stopVault()

	bound, stop := context.WithTimeout(context.WithoutCancel(ctx), ShutdownBound)
	defer stop()
	var failures []error
	for _, closer := range u.closers {
		failures = append(failures, closer.Close(bound))
	}
	return errors.Join(failures...)
}
