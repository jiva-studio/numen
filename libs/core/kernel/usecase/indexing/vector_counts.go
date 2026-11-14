package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
)

// VectorCounts counts the chunks of a vault that can carry a vector and the ones that carry one, under the
// model the vault is embedded with.
type VectorCounts struct {
	embedding *Embedding
	recorder  indexport.ModelRecorder
}

// NewVectorCounts creates the count over the embedding of a window, and records the model it counts under.
func NewVectorCounts(embedding *Embedding, recorder indexport.ModelRecorder) VectorCounts {
	return VectorCounts{embedding: embedding, recorder: recorder}
}

// Execute answers ErrNoProgress where the index cannot report how far the embedding is.
func (u VectorCounts) Execute(ctx context.Context, v domain.Vault) (held, embedded int64, err error) {
	if !u.embedding.HasProgress() {
		return 0, 0, ErrNoProgress
	}
	name, recipe := u.embedding.Describe()
	u.recorder.RecordModel(name, recipe)
	return u.embedding.CountProgress(ctx, v.ID, recipe)
}
