package vaultport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// VaultWatcher follows one vault for changes the application did not make. The
// filesystem is one implementation; a channel a test writes to is another.
type VaultWatcher interface {
	// WatchVault reports the paths of sources that change, folded, until ctx is
	// done. Which kind each is comes from a stat of the path.
	//
	// `lost` signals that the vault has to be read again: more changed at once than
	// could be reported, or something was removed whose contents cannot be queried.
	//
	// Two channels: an implementer does not need to name the interface, and Go
	// checks conformance implicitly.
	WatchVault(ctx context.Context, v domain.Vault) (changes <-chan []string, lost <-chan struct{}, err error)
}
