package embedding

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/internal/embedding"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// getIndexLaneLimit is how many calls the index side may have in flight on a
// model that runs passes forward passes at once: one fewer, and never fewer
// than one. A question put to the same model always finds a pass free but the
// one it may have to wait for, and a model of one pass lets both sides through.
func getIndexLaneLimit(passes int) int {
	return max(passes-1, 1)
}

// indexLane is the index side of one model that questions are asked of too. It
// holds at most getIndexLaneLimit calls in flight, and the question side is not
// held by it.
type indexLane struct {
	activeItem *embedding.Embedder
	waiting    embeddingport.Embedder

	once  sync.Once
	slots chan struct{}
}

func newIndexLane(target *embedding.Embedder) *indexLane {
	return &indexLane{activeItem: target, waiting: target.GetWaitingEmbedder()}
}

func (l *indexLane) GetModel() embeddingport.EmbeddingModel { return l.activeItem.GetModel() }

// Close lets go of the model both sides ask.
func (l *indexLane) Close() error { return l.activeItem.Close() }

// Concurrency is how many calls the index side runs at once.
func (l *indexLane) Concurrency() int {
	return getIndexLaneLimit(l.activeItem.Concurrency())
}

// Wait blocks until the model turns up, and says what stopped it.
func (l *indexLane) Wait(ctx context.Context) error { return l.activeItem.Wait(ctx) }

// Embed waits for the model and for a free slot in the lane, and returns the
// slot when the call returns or ctx ends.
func (l *indexLane) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := l.activeItem.Wait(ctx); err != nil {
		return nil, err
	}
	l.once.Do(func() { l.slots = make(chan struct{}, l.Concurrency()) })
	select {
	case l.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-l.slots }()
	return l.waiting.Embed(ctx, texts)
}
