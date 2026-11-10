package secret

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// CredentialsLocator is a key store that keeps its values in a file when the machine has no keychain.
type CredentialsLocator interface {
	// GetCredentialsPath returns the file the values are kept in, or the empty text while the
	// system keychain is in use.
	GetCredentialsPath(ctx context.Context) string
}

// GetCredentialsPath returns the function that answers where the values of the key store are kept: the file
// of a store that keeps them in one, else the empty text.
func GetCredentialsPath(store pluginport.KeyStore) func(ctx context.Context) string {
	return func(ctx context.Context) string {
		locator, isLocator := store.(CredentialsLocator)
		if !isLocator {
			return ""
		}
		return locator.GetCredentialsPath(ctx)
	}
}
