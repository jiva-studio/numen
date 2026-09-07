package vault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

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

// addresses is what the notes of this vault point at, taken before any of them
// are removed. Nothing is asked where the walk found every note still there.
func (u Scan) addresses(
	ctx context.Context, v domain.Vault, gone []string,
) ([]string, error) {
	if len(gone) == 0 || u.Derived == nil {
		return nil, nil
	}
	held, err := u.Known.Addresses(ctx, v.ID)
	if err != nil {
		return nil, fmt.Errorf("read index: %w", err)
	}
	return held, nil
}

// sweep takes out what was fetched for an address no note points at any more.
//
// What was fetched is named by the address and shared by every note carrying
// it, so the question is asked of the vault and not of the note that went: two
// notes on one video keep it while either of them stands.
func (u Scan) sweep(
	ctx context.Context, v domain.Vault, held port.DerivedStore, was []string,
) error {
	if held == nil || len(was) == 0 {
		return nil
	}
	stands, err := u.Known.Addresses(ctx, v.ID)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	for _, address := range was {
		if slices.Contains(stands, address) {
			continue
		}
		for _, name := range text.AddressNames(text.Fingerprint([]byte(address))) {
			if err := held.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("remove %s: %w", name, err)
			}
		}
	}
	return nil
}

// unasked reaches the address of every link note nothing has been fetched for,
// and answers how many were reached.
//
// It is `importing.fetch_unasked`, which is off: reaching off the machine is a
// gesture, and a note somebody wrote in another editor is not one. Turned on,
// a walk finds a link note the way it finds any other and what is at its
// address is there when the person opens it.
//
// An address that would not answer is that note's trouble. The walk found every
// file it found, and one site refusing does not unsay it.
func (u Scan) unasked(ctx context.Context, v domain.Vault, held port.DerivedStore) int {
	if u.Fetches == nil || u.Types == nil || held == nil {
		return 0
	}
	pointing, err := u.Types.OfType(ctx, v.ID, domain.TypeLink)
	if err != nil {
		return 0
	}
	fetched := 0
	for _, path := range pointing {
		if ctx.Err() != nil {
			return fetched
		}
		if err := u.Fetches(ctx, v, path); err == nil {
			fetched++
		}
	}
	return fetched
}
