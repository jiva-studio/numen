package indexing

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNoProgress is what the count of embedded chunks answers when the index cannot report it.
var ErrNoProgress = errors.New("index progress is unknown")

// Embedding is the embedder a window indexes its vault with, and how far the vault has been embedded
// under its model. Either may be missing: a window without an embedder indexes words alone.
type Embedding struct {
	embedder embeddingport.Embedder
	watcher  embeddingport.ModelWatcher
	progress indexport.IndexProgress
}

// NewEmbedding returns the embedding of an embedder, the watcher of its model and the count of what it has
// embedded. An embedder that never changes its model has no watcher.
func NewEmbedding(embedder embeddingport.Embedder, watcher embeddingport.ModelWatcher, progress indexport.IndexProgress) *Embedding {
	return &Embedding{embedder: embedder, watcher: watcher, progress: progress}
}

// GetEmbedder returns the embedder, and none for a window without one.
func (e *Embedding) GetEmbedder() embeddingport.Embedder {
	if e == nil {
		return nil
	}
	return e.embedder
}

// Describe returns the display name and the recipe of the model, and nothing without an embedder.
func (e *Embedding) Describe() (name, recipe string) {
	if e == nil || e.embedder == nil {
		return "", ""
	}
	model := e.embedder.GetModel()
	return model.String(), model.Recipe()
}

// CountProgress returns how many chunks of the vault can carry a vector under the recipe and how many
// carry one.
func (e *Embedding) CountProgress(ctx context.Context, vaultID domain.VaultID, recipe string) (total, embedded int64, err error) {
	if e == nil || e.progress == nil {
		return 0, 0, ErrNoProgress
	}
	return e.progress.GetProgress(ctx, vaultID, recipe)
}

// HasProgress reports whether the embedded chunks can be counted.
func (e *Embedding) HasProgress() bool { return e != nil && e.progress != nil }

// OnModelChange calls fn each time the model of the embedder changes, with whether the change moved
// the size of a chunk. An embedder that never changes its model calls nothing.
func (e *Embedding) OnModelChange(fn func(isChunkSizeChanged bool)) {
	if e == nil || e.embedder == nil || e.watcher == nil {
		return
	}
	chunkTokens := e.embedder.GetModel().ChunkTokens
	e.watcher.OnModelChange(func() {
		tokens := e.embedder.GetModel().ChunkTokens
		changed := tokens != chunkTokens
		chunkTokens = tokens
		fn(changed)
	})
}
