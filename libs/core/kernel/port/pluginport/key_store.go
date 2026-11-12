package pluginport

import (
	"context"
)

// KeyStore stores the keys that authorize a service for a user, such as an API
// key, by name.
type KeyStore interface {
	// GetKey returns the key stored under the name, or plugin.ErrNoKey.
	GetKey(ctx context.Context, name string) (string, error)
	// SetKey stores the key under the name and replaces the one already there.
	SetKey(ctx context.Context, name, key string) error
	// DeleteKey removes the key stored under the name, or fails with plugin.ErrNoKey.
	DeleteKey(ctx context.Context, name string) error
}

// CredentialsLocator tells where a key store keeps its values when the machine has no keychain.
type CredentialsLocator interface {
	// GetCredentialsPath returns the file the values are kept in, or the empty text while the
	// system keychain is in use.
	GetCredentialsPath(ctx context.Context) string
}
