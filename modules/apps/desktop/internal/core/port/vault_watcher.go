package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// VaultWatcher follows one vault for changes the application did not make. The
// filesystem is one implementation; a channel a test writes to is another.
type VaultWatcher interface {
	// Watch reports the paths of notes that change, folded, until ctx is done.
	//
	// `lost` says the vault has to be read again rather than followed: more
	// changed at once than could be reported, or something went that cannot be
	// asked what it held.
	//
	// Two channels rather than one value, so that whoever implements this never
	// has to name this interface: Go checks the fit without either side saying
	// so, and a shared struct would have to live somewhere both can import.
	Watch(ctx context.Context, v domain.Vault) (changes <-chan []string, lost <-chan struct{}, err error)
}
