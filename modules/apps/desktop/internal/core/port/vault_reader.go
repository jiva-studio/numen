package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// VaultReader is one vault as seen from outside: something that can be walked
// and read. The filesystem is one implementation; a fixture in memory is
// another.
type VaultReader interface {
	// Walk reports every file that is a candidate for indexing, in unspecified
	// order. The service folder is not reported.
	Walk(ctx context.Context, fn func(domain.FileRef) error) error
	// Read returns the bytes of one file, addressed by a path a walk reported.
	Read(ctx context.Context, path string) ([]byte, error)
}

// VaultReaders opens a reader for a given vault. A use case is handed this
// rather than a reader, because which vault it works on is decided while it
// runs, not when it is built.
type VaultReaders interface {
	Open(v domain.Vault) (VaultReader, error)
}
