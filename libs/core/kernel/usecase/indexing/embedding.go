package indexing

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNoProgress is what the count of indexed chunks answers when the index cannot report it.
var ErrNoProgress = errors.New("index progress is unknown")

// Embedding is the search source a window indexes its vault with, and how far the vault has been indexed
// under its recipe. Either may be missing: a window without a source indexes words alone.
type Embedding struct {
	source   searchsourceport.Source
	watcher  searchsourceport.DescriptionWatcher
	progress indexport.IndexProgress
}

// NewEmbedding returns the embedding of a search source, the watcher of its description and the count of
// what it has indexed. A source whose description never changes has no watcher.
func NewEmbedding(source searchsourceport.Source, watcher searchsourceport.DescriptionWatcher, progress indexport.IndexProgress) *Embedding {
	return &Embedding{source: source, watcher: watcher, progress: progress}
}

// GetSource returns the search source, and none for a window without one.
func (e *Embedding) GetSource() searchsourceport.Source {
	if e == nil {
		return nil
	}
	return e.source
}

// Describe returns the display name and the recipe of the source, and nothing without a source or for a
// source that has not described itself.
func (e *Embedding) Describe() (name, recipe string) {
	if e == nil || e.source == nil {
		return "", ""
	}
	described := e.source.Describe()
	if !described.IsDescribed() {
		return "", ""
	}
	return described.String(), described.Recipe()
}

// CountProgress returns how many chunks of the vault can carry a vector under the recipe and how many
// carry one.
func (e *Embedding) CountProgress(ctx context.Context, vaultID domain.VaultID, recipe string) (total, embedded int64, err error) {
	if e == nil || e.progress == nil {
		return 0, 0, ErrNoProgress
	}
	return e.progress.GetProgress(ctx, vaultID, recipe)
}

// HasProgress reports whether the indexed chunks can be counted.
func (e *Embedding) HasProgress() bool { return e != nil && e.progress != nil }

// OnModelChange calls fn each time the description of the source changes, with whether the change moved
// the size of a chunk. A source whose description never changes calls nothing.
func (e *Embedding) OnModelChange(fn func(isChunkSizeChanged bool)) {
	if e == nil || e.source == nil || e.watcher == nil {
		return
	}
	chunkTokens := e.source.Describe().ChunkTokens
	e.watcher.OnDescriptionChange(func() {
		tokens := e.source.Describe().ChunkTokens
		changed := tokens != chunkTokens
		chunkTokens = tokens
		fn(changed)
	})
}
