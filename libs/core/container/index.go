package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/adapter/index"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Index is the cache, seen as the ports the core asks for. The concrete adapter
// stays inside this package: a command should not have to know that the answer
// is SQLite in order to ask for somewhere to put a note.
type Index struct {
	db   *index.DB
	path string
	// walks is the turn each vault takes to be walked into this index. It is
	// the index's because a walk is what writes into one: two indexes are
	// walked at once, and one index is walked a vault at a time.
	walks vault.Walks
	// ledger is shared by every refresh of this index.
	ledger *vault.Ledger
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
	return &Index{db: db, path: path, ledger: vault.NewLedger()}, nil
}

func (i *Index) Close() error { return i.db.Close() }

// Walks is the turns the vaults walked into this index take.
func (i *Index) Walks() *vault.Walks { return &i.walks }

// Level brings the notes at the paths given up to date in the index. Whatever
// writes a note calls it with the paths it touched, so what it wrote is
// findable by the time the write returns.
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
	refresh.Ledger = db.ledger
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
	return vaultwatch.Watcher{Options: c.vaultOptions()}.File(ctx, path)
}

// FitVectors makes the vector index hold vectors of the width given, filled
// from what the recipe has already bought.
func (i *Index) FitVectors(ctx context.Context, dims int, recipe string) error {
	return i.db.FitVectors(ctx, dims, recipe)
}

// DeleteVectorsOfOtherRecipes takes out the vectors kept under any recipe but the one in
// use, and says how many went.
func (i *Index) DeleteVectorsOfOtherRecipes(ctx context.Context, recipe string) (int64, error) {
	return i.db.DeleteOtherRecipeVectors(ctx, recipe)
}

// Compact returns the space the index no longer holds.
func (i *Index) Compact(ctx context.Context) error { return i.db.Compact(ctx) }

func (i *Index) Vaults() vaultport.VaultRepository { return i.db.Vaults() }
func (i *Index) Notes() noteport.NoteRepository    { return i.db.Notes() }
func (i *Index) Queries() noteport.NoteQueries     { return i.db.NoteQueries() }

// Passages is the two indexes a search runs over. They read, so a search answers
// while a scan is still writing.
func (i *Index) Passages() sourceport.PassageQueries { return i.db.ChunkQueries() }

// Progress is how far chunking and embedding have got, for a window to say so.
func (i *Index) Progress() indexport.IndexProgress { return i.db.ChunkQueries() }

// Sources holds what has text and what was made from it. One type answers all
// four ports: they divide the same tables by what asks, not by where the rows
// are.
func (i *Index) Sources() sourceport.SourceRepository       { return i.db.Sources() }
func (i *Index) SourcesKnown() sourceport.SourceQueries     { return i.db.Sources() }
func (i *Index) Vectors() embeddingport.VectorRepository    { return i.db.Sources() }
func (i *Index) VectorQueries() embeddingport.VectorQueries { return i.db.Sources() }

// Maintenance is how the index is told that it has changed wholesale.
func (i *Index) Maintenance() indexport.IndexMaintenance { return i.db.Maintenance() }

// Path is where the database file is, which a load test needs in order to say
// how large the index got.
func (i *Index) Path() string { return i.path }

// Links answers what points where.
func (i *Index) Links() noteport.LinkQueries { return i.db.NoteQueries() }

// Problems reports what a scan could not act on.
func (i *Index) Problems() recognitionport.ProblemQueries { return i.db.NoteQueries() }
