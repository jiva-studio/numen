package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Index is the cache, seen as the ports the core asks for. The concrete adapter
// stays inside this package: a command should not have to know that the answer
// is SQLite in order to ask for somewhere to put a note.
type Index struct {
	db   *index.DB
	path string
}

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
	return &Index{db: db, path: path}, nil
}

func (i *Index) Close() error { return i.db.Close() }

// Level brings the notes at the paths given up to date in the index. Whatever
// writes a note calls it with the paths it touched, so what it wrote is
// findable by the time the write returns.
func (c Config) Level(db *Index) func(ctx context.Context, v domain.Vault, paths []string) error {
	refresh := vault.Refresh{
		Readers: c.VaultReaders(),
		Notes:   db.NotesCutAt(c.Chunking()),
		Known:   db.SourcesKnown(),
		Sources: db.Sources(),
	}
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

// Moves reports each time the index file changes. What a vault holds is the
// index's answer, and another window writing the index changes it.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	path, err := c.indexPath()
	if err != nil {
		return nil, err
	}
	return filesystem.Watcher{Options: c.VaultOptions()}.File(ctx, path)
}

// FitVectors makes the vector index hold vectors of the width given, filled
// from what the recipe has already bought.
func (i *Index) FitVectors(ctx context.Context, dims int, recipe string) error {
	return i.db.FitVectors(ctx, dims, recipe)
}

func (i *Index) Vaults() port.VaultRepository { return i.db.Vaults() }
func (i *Index) Notes() port.NoteRepository   { return i.db.Notes() }
func (i *Index) Queries() port.NoteQueries    { return i.db.NoteQueries() }

// Passages is the two indexes a search runs over. They read, so a search answers
// while a scan is still writing.
func (i *Index) Passages() port.PassageQueries { return i.db.ChunkQueries() }

// Progress is how far chunking and embedding have got, for a window to say so.
func (i *Index) Progress() port.IndexProgress { return i.db.ChunkQueries() }

// Sources holds what has text and what was made from it. One type answers all
// four ports: they divide the same tables by what asks, not by where the rows
// are.
func (i *Index) Sources() port.SourceRepository   { return i.db.Sources() }
func (i *Index) SourcesKnown() port.SourceQueries { return i.db.Sources() }
func (i *Index) Vectors() port.VectorRepository   { return i.db.Sources() }
func (i *Index) VectorsOwing() port.VectorQueries { return i.db.Sources() }

// Maintenance is how the index is told that it has changed wholesale.
func (i *Index) Maintenance() port.IndexMaintenance { return i.db.Statistics() }

// Path is where the database file is, which a load test needs in order to say
// how large the index got.
func (i *Index) Path() string { return i.path }

// Links answers what points where.
func (i *Index) Links() port.LinkQueries { return i.db.NoteQueries() }

// Problems reports what a scan could not act on.
func (i *Index) Problems() port.ProblemQueries { return i.db.NoteQueries() }
