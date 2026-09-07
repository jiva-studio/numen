package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/fetch"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// Fetcher is what reaches an address on this machine, and nothing where the
// machine holds neither of the tools that reach one.
//
// A build with no fetcher is a build that cannot import: the run is answered
// that it cannot be done here, and the window offers it nowhere from then on.
func (c Config) Fetcher(ctx context.Context) port.Fetcher {
	fetcher, err := fetch.New(ctx, c.Fetching)
	if errors.Is(err, fetch.ErrNoTool) {
		return nil
	}
	if err != nil {
		c.trouble(err)
		return nil
	}
	return fetcher
}

// ImportURL is one link note's address, fetched into the vault, with the note
// cut again as soon as what was fetched is written. A machine holding neither
// tool has none.
func (c Config) ImportURL(ctx context.Context, db *Index, by port.Fetcher) source.ImportURL {
	level := c.Level(db)
	notes := c.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), level)
	return source.ImportURL{
		Readers:   c.VaultReaders(),
		Derived:   c.DerivedStores(),
		By:        by,
		CopyUnder: c.Fetching.CopyBytes(),
		Languages: c.Fetching.Captions,
		Automatic: c.Fetching.Automatic(),
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			return level(ctx, v, []string{path})
		},
		Names: func(ctx context.Context, v domain.Vault, path, title string) (string, error) {
			named, err := notes.Rename.Execute(ctx, v, path, title)
			return named.Path, err
		},
	}
}
