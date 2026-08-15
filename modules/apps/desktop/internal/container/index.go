package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Index is the cache, seen as the ports the core asks for. The concrete adapter
// stays inside this package: a command should not have to know that the answer
// is SQLite in order to ask for somewhere to put a note.
type Index struct{ db *index.DB }

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
	return &Index{db: db}, nil
}

func (i *Index) Close() error { return i.db.Close() }

func (i *Index) Vaults() port.VaultRepository { return i.db.Vaults() }
func (i *Index) Notes() port.NoteRepository   { return i.db.Notes() }
func (i *Index) Queries() port.NoteQueries    { return i.db.NoteQueries() }
