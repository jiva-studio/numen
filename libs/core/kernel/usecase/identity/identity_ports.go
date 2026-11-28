package identity

import "context"

// DeviceName gives the name a device is listed under in the device chain.
type DeviceName interface {
	// GetInstanceName returns the human-readable name of this machine.
	GetInstanceName() string
}

// ChainStore keeps the device chain of this device in a file next to the key store entry.
type ChainStore interface {
	// Read returns the bytes of the chain, or identity.ErrNoChain when there is none.
	Read(ctx context.Context) ([]byte, error)
	Write(ctx context.Context, chain []byte) error
	Delete(ctx context.Context) error
}
