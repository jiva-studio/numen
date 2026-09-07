package container

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/fetch"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// Fetcher is what reaches an address on this machine.
//
// A page needs no tool, so every build has one. Which addresses it reaches is
// the machine's: a video is asked of yt-dlp, and a machine without it says so
// when one is asked for.
func (c Config) Fetcher(ctx context.Context) port.Fetcher {
	fetcher, err := fetch.New(ctx, c.Fetching)
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
		Readers:     c.VaultReaders(),
		Derived:     c.DerivedStores(),
		By:          by,
		CopyMaxSize: c.Fetching.CopyBytes(),
		ToVault:     c.Fetching.ToVault(),
		Writers:     c.VaultWriters(),
		Languages:   c.Fetching.Captions,
		Automatic:   c.Fetching.Automatic(),
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			return level(ctx, v, []string{path})
		},
		Names: func(ctx context.Context, v domain.Vault, path, title string) (string, error) {
			named, err := notes.Rename.Execute(ctx, v, path, title)
			return named.Path, err
		},
	}
}

// FetchesUnasked reaches the address of a link note the vault holds nothing
// fetched for, as a walk finds it. It is nothing where the settings do not ask
// for it, which is where the settings say nothing.
//
// The fetcher and what imports through it are made on the first note that needs
// them: a walk of a vault holding no link note asks this machine for nothing.
func (c Config) FetchesUnasked(db *Index) func(context.Context, domain.Vault, string) error {
	if !c.Fetching.Unasked() {
		return nil
	}
	var once sync.Once
	var importing source.ImportURL
	var able bool
	return func(ctx context.Context, v domain.Vault, path string) error {
		once.Do(func() {
			by := c.Fetcher(ctx)
			if by == nil {
				return
			}
			importing, able = c.ImportURL(ctx, db, by), true
		})
		if !able {
			return nil
		}
		_, err := importing.Execute(ctx, v, path)
		return err
	}
}
