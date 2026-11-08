package vault

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/core/adapter/index"
	"github.com/jiva-studio/numen/libs/core/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	vaultusecase "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Index is the cache, exposed as the ports that the core uses.
type Index = vaultusecase.Index

// ErrNoIndexPath is a configuration that names no index database.
var ErrNoIndexPath = errors.New("the configuration names no index database: the application sets IndexPath")

// ResolveIndexPath ensures the parent folder of the index database exists and returns its path.
func (c Config) ResolveIndexPath() (string, error) {
	if c.IndexPath == "" {
		return "", ErrNoIndexPath
	}
	if err := os.MkdirAll(filepath.Dir(c.IndexPath), 0o755); err != nil {
		return "", err
	}
	return c.IndexPath, nil
}

// OpenIndex opens the cache. The caller closes it.
func (c Config) OpenIndex(ctx context.Context) (*Index, error) {
	path, err := c.ResolveIndexPath()
	if err != nil {
		return nil, err
	}
	db, err := index.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	return vaultusecase.NewIndex(vaultusecase.IndexPorts{
		Backend:       db,
		Path:          path,
		Vaults:        db.Vaults(),
		Notes:         db.Notes(),
		Queries:       db.NoteQueries(),
		Passages:      db.ChunkQueries(),
		Progress:      db.ChunkQueries(),
		Stats:         db.ChunkQueries(),
		Sources:       db.Sources(),
		SourcesKnown:  db.Sources(),
		Vectors:       db.Sources(),
		VectorQueries: db.Sources(),
		Maintenance:   db.Maintenance(),
		Links:         db.NoteQueries(),
		Problems:      db.NoteQueries(),
		OpenTabs:      db.OpenTabs(),
		TextVersions:  version.NewIndexVersions(db.TextVersions()),
		SearchFloors:  db.SearchFloors(),
		SearchSamples: db.SearchSamples(),
		Nearest:       db.ChunkQueries(),
	}), nil
}

// UpdateIndex updates the notes at the paths given up to date in the index.
func (c Config) UpdateIndex(db *Index) note.IndexUpdater {
	refresh := vaultusecase.NewRefresh(
		c.VaultReaders(),
		db.Vaults(),
		db.Notes(),
		db.SourcesKnown(),
		db.Sources(),
		c.Clock,
	)
	refresh.Cut = c.CutNotes
	refresh.Ledger = db.Ledger()
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

// Moves reports each time the index file changes.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	path, err := c.ResolveIndexPath()
	if err != nil {
		return nil, err
	}
	return vaultwatch.Watcher{Options: c.vaultOptions(), Hub: c.Watches}.File(ctx, path)
}
