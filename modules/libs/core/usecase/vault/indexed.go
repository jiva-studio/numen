package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// store is where this vault's fetches are kept, and nothing where the run was
// given no store or cannot reach it. A walk without one indexes every note as
// the prose in its file.
func store(stores port.DerivedStores, v domain.Vault) port.DerivedStore {
	if stores == nil {
		return nil
	}
	held, err := stores.Open(v)
	if err != nil {
		return nil
	}
	return held
}

// indexed is one note as it goes into the index, carrying what was fetched for
// the address it points at.
//
// A link note is searched as what its person wrote and what is at that address
// together, so the two reach the index in one breath. Every other note is
// itself.
func indexed(ctx context.Context, held port.DerivedStore, n domain.Note) domain.IndexedNote {
	if held == nil || n.Type != domain.TypeLink {
		return domain.IndexedNote{Note: n}
	}
	at, wrong := domain.ReadAddress(n.Frontmatter)
	if len(wrong) > 0 {
		return domain.IndexedNote{Note: n}
	}
	words, producer, err := text.Fetched(ctx, held, text.Fingerprint([]byte(at.URL)))
	if err != nil {
		// The store is a folder on the person's disk and they may empty it. The
		// note is indexed as its prose, and the next walk finds what is there.
		return domain.IndexedNote{Note: n}
	}
	return domain.IndexedNote{Note: n, Artifact: domain.Artifact{Producer: producer, Text: words}}
}
