package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// VaultWatcher follows one vault for changes the application did not make. The
// filesystem is one implementation; a channel a test writes to is another.
type VaultWatcher interface {
	// Watch reports the paths of sources that change, folded, until ctx is
	// done. Which kind each is comes from a stat of the path.
	//
	// `lost` says the vault has to be read again: more changed at once than
	// could be reported, or something went that cannot be asked what it held.
	//
	// Two channels: whoever implements this never has to name the interface,
	// and Go checks the fit without either side saying so.
	Watch(ctx context.Context, v domain.Vault) (changes <-chan []string, lost <-chan struct{}, err error)
}
