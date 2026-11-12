package indexing_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

// announcing is an embedder whose model changes when its plugin says so.
type announcing struct {
	model     searchsource.Description
	listeners []func()
}

func (a *announcing) Describe() searchsource.Description { return a.model }

func (*announcing) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (*announcing) Close() error { return nil }

func (*announcing) Concurrency() int { return 1 }

func (a *announcing) OnModelChange(fn func()) { a.listeners = append(a.listeners, fn) }

func (a *announcing) change(model searchsource.Description) {
	a.model = model
	for _, fn := range a.listeners {
		fn()
	}
}

// silent is an embedder that cannot announce a change.
type silent struct{}

func (silent) Describe() searchsource.Description { return searchsource.Description{} }

func (silent) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (silent) Close() error { return nil }

func (silent) Concurrency() int { return 1 }

func newChunking(tokens int) *announcing {
	return &announcing{model: searchsource.Description{Name: "first", Width: 8, SourceRecipe: "mean", ChunkTokens: tokens}}
}

func watchChunkSizes(embedder *announcing, sizes ...int) []bool {
	var changed []bool
	indexing.NewEmbedding(sourcefake.Wrap(embedder), sourcefake.Wrap(embedder), nil).OnModelChange(func(isChunkSizeChanged bool) {
		changed = append(changed, isChunkSizeChanged)
	})
	for _, tokens := range sizes {
		next := embedder.model
		next.ChunkTokens = tokens
		embedder.change(next)
	}
	return changed
}

func TestANewChunkSizeIsAnnouncedAsAChange(t *testing.T) {
	got := watchChunkSizes(newChunking(256), 128)

	if len(got) != 1 || !got[0] {
		t.Errorf("announced %v for 256 then 128, want one change", got)
	}
}

func TestAChunkSizeThatStaysIsNotAChange(t *testing.T) {
	embedder := newChunking(256)
	var changed []bool
	indexing.NewEmbedding(sourcefake.Wrap(embedder), sourcefake.Wrap(embedder), nil).OnModelChange(func(isChunkSizeChanged bool) {
		changed = append(changed, isChunkSizeChanged)
	})

	embedder.change(embedder.model)
	other := embedder.model
	other.Name = "second"
	embedder.change(other)

	if len(changed) != 2 || changed[0] || changed[1] {
		t.Errorf("announced %v for a model that kept its chunk size, want no change", changed)
	}
}

func TestEachChangeOfTheChunkSizeIsComparedWithTheOneBefore(t *testing.T) {
	got := watchChunkSizes(newChunking(256), 128, 128, 256)

	want := []bool{true, false, true}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("announced %v for 256, 128, 128, 256; want %v", got, want)
	}
}

func TestAChunkSizeThatBecomesKnownIsAChange(t *testing.T) {
	got := watchChunkSizes(newChunking(0), 256)

	if len(got) != 1 || !got[0] {
		t.Errorf("announced %v for a size that became known, want one change", got)
	}
}

func TestAnEmbedderThatAnnouncesNothingIsNotFollowed(t *testing.T) {
	var absent *indexing.Embedding
	absent.OnModelChange(func(bool) { t.Error("a window without an embedder announced a change") })
	indexing.NewEmbedding(sourcefake.Wrap(silent{}), nil, nil).OnModelChange(func(bool) { t.Error("a silent embedder announced a change") })
}
