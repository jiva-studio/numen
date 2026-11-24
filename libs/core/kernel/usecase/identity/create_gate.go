package identity

import "context"

// CreateGate lets one creation of the identity run at a time over a key store. A creation that waits
// for the gate stops when its context ends.
type CreateGate chan struct{}

// NewCreateGate returns an open gate.
func NewCreateGate() CreateGate { return make(CreateGate, 1) }

func (g CreateGate) acquire(ctx context.Context) (func(), error) {
	select {
	case g <- struct{}{}:
		return func() { <-g }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
