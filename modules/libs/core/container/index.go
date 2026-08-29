package container

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

// ReadIndex is the cache seen by a process that only asks it questions. It
// offers no repository, so nothing reached through it can write.
type ReadIndex struct {
	db *index.Reading
}

// OpenIndexToRead opens the cache for asking alone.
//
// An index that is not there is not made on disk: it is built by the
// application that scans, and until that has run once every vault reads as one
// nothing has read yet.
func (c Config) OpenIndexToRead(ctx context.Context) (*ReadIndex, error) {
	path, err := c.indexPath()
	if err != nil {
		return nil, err
	}
	db, err := index.OpenToRead(ctx, path)
	if errors.Is(err, fs.ErrNotExist) {
		db, err = index.OpenNothing(ctx)
	}
	if err != nil {
		return nil, err
	}
	return &ReadIndex{db: db}, nil
}

func (i *ReadIndex) Close() error { return i.db.Close() }

// Moves reports each time the index changes underneath a process that only
// reads it. What a vault holds is the index's answer, and the answer changes
// when the application that scans writes one.
func (c Config) Moves(ctx context.Context) (<-chan struct{}, error) {
	path, err := c.indexPath()
	if err != nil {
		return nil, err
	}
	return filesystem.Watcher{Options: c.VaultOptions()}.File(ctx, path)
}

func (i *ReadIndex) Queries() port.NoteQueries { return i.db.NoteQueries() }
func (i *ReadIndex) Links() port.LinkQueries   { return i.db.NoteQueries() }

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

// Progress is how far cutting and embedding have got, for a window to say so.
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
