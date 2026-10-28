package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/adapter/index"
	"github.com/jiva-studio/numen/libs/core/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Index is the cache, seen as the ports the core asks for.
type Index = vault.Index

// OpenIndex opens the cache. Closing it belongs to the caller, which is what
// knows when it is finished.
func (c Config) OpenIndex(ctx context.Context) (*Index, error) {
	path, err := c.indexPath()
	if err != nil {
		return nil, err
	}
	db, err := index.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	return vault.NewIndex(vault.IndexPorts{
		Backend:       db,
		Path:          path,
		Vaults:        db.Vaults(),
		Notes:         db.Notes(),
		Queries:       db.NoteQueries(),
		Passages:      db.ChunkQueries(),
		Progress:      db.ChunkQueries(),
		Sources:       db.Sources(),
		SourcesKnown:  db.Sources(),
		Vectors:       db.Sources(),
		VectorQueries: db.Sources(),
		Maintenance:   db.Maintenance(),
		Links:         db.NoteQueries(),
		Problems:      db.NoteQueries(),
		OpenTabs:      db.OpenTabs(),
		TextVersions:  version.NewIndexVersions(db.TextVersions()),
	}), nil
}

// Level brings the notes at the paths given up to date in the index.
func (c Config) Level(db *Index) note.Levels {
	refresh := vault.NewRefresh(
		c.VaultReaders(),
		db.Vaults(),
		db.Notes(),
		db.SourcesKnown(),
		db.Sources(),
		c.Clock(),
	)
	refresh.Cut = c.CutNotes()
	refresh.Ledger = db.Ledger()
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

// Moves reports each time the index file changes.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	path, err := c.indexPath()
	if err != nil {
		return nil, err
	}
	return vaultwatch.Watcher{Options: c.vaultOptions()}.File(ctx, path)
}
