package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// isUnclaimed says whether nobody holds a name, by taking it and letting it go.
func isUnclaimed(ctx context.Context, store port.DerivedStore, name string) bool {
	release, err := store.ClaimName(ctx, name)
	if err != nil {
		return false
	}
	release()
	return true
}

// readFirst is what the store holds under the first of these names, and nothing where it holds none.
func readFirst(ctx context.Context, store port.DerivedStore, names ...string) ([]byte, error) {
	for _, name := range names {
		raw, err := store.Read(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return nil, nil
}
