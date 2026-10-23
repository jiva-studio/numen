package artifacts

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// isUnclaimed says whether nobody holds a name, by taking it and letting it go.
func isUnclaimed(ctx context.Context, store vaultport.DerivedStore, name domain.ArtifactName) bool {
	release, err := store.ClaimArtifact(ctx, name)
	if err != nil {
		return false
	}
	release()
	return true
}

// readFirst is what the store holds under the first of these names, and nothing where it holds none.
func readFirst(ctx context.Context, store vaultport.DerivedStore, names ...domain.ArtifactName) ([]byte, error) {
	for _, name := range names {
		raw, err := store.ReadArtifact(ctx, name)
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
