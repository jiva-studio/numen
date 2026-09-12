package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/download"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// Downloader is what reaches an address on this machine.
//
// A page needs no tool, so every build has one. Which addresses it reaches is
// the machine's: a video is asked of yt-dlp, and a machine without it says so
// when one is asked for.
func (c Config) Downloader(ctx context.Context) port.Downloader {
	downloader, err := download.New(ctx, c.Importing)
	if err != nil {
		c.handleError(err)
		return nil
	}
	return downloader
}

// ImportURL is one link note's address, downloaded into the vault, with the
// note cut again as soon as what came back is written. A machine holding
// neither tool has none.
func (c Config) ImportURL(ctx context.Context, db *Index, by port.Downloader) source.ImportURL {
	level := c.Level(db)
	notes := c.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), level)
	return source.ImportURL{
		Readers:     c.VaultReaders(),
		Derived:     c.GetDerivedStores(),
		By:          by,
		CopyMaxSize: c.Importing.CopyBytes(),
		ToVault:     c.Importing.KeepsCopiesInVault(),
		Writers:     c.VaultWriters(),
		Languages:   c.Importing.Captions,
		Automatic:   c.Importing.AllowsAutomaticCaptions(),
		Cut: func(ctx context.Context, v domain.Vault, path string) error {
			return level(ctx, v, []string{path})
		},
		Names: func(ctx context.Context, v domain.Vault, path, title string) (string, error) {
			named, err := notes.Rename.Execute(ctx, v, path, title)
			return named.Path, err
		},
	}
}
