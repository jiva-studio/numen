package port

import (
	"context"
	"errors"
)

var (
	// ErrNoKey means the store holds no key under the name.
	ErrNoKey = errors.New("no key is stored under that name")

	// ErrNoKeyStore means this machine has no store to keep a key in. A key is
	// never kept anywhere else, and the caller is told so.
	ErrNoKeyStore = errors.New("this machine has no key store")
)

// KeyStore keeps the keys that open a service for a person, such as an API key,
// by name.
type KeyStore interface {
	// GetKey returns the key stored under the name, or ErrNoKey.
	GetKey(ctx context.Context, name string) (string, error)
	// SetKey stores the key under the name and replaces the one already there.
	SetKey(ctx context.Context, name, key string) error
	// DeleteKey removes the key stored under the name, or fails with ErrNoKey.
	DeleteKey(ctx context.Context, name string) error
}
