package editor

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// announcing is an embedder whose model changes when its plugin says so.
type announcing struct {
	model     embeddingport.EmbeddingModel
	listeners []func()
}

func (a *announcing) GetModel() embeddingport.EmbeddingModel { return a.model }

func (*announcing) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (*announcing) Close() error { return nil }

func (*announcing) Concurrency() int { return 1 }

func (a *announcing) OnModelChange(fn func()) { a.listeners = append(a.listeners, fn) }

func (a *announcing) change(model embeddingport.EmbeddingModel) {
	a.model = model
	for _, fn := range a.listeners {
		fn()
	}
}

// The model the window shows follows the model the embedder announces.
func TestTheRecordedModelFollowsTheModelTheEmbedderAnnounces(t *testing.T) {
	embedder := &announcing{model: embeddingport.EmbeddingModel{Name: "first", Dimensions: 8, Pooling: "mean"}}
	state := &State{}
	state.Indexing.Model.Store(embedder.model.String())
	state.Indexing.Recipe.Store(embedder.model.Recipe())

	followModel(embedder, state, nil, nil)
	second := embeddingport.EmbeddingModel{Name: "second", Dimensions: 16, Pooling: "mean"}
	embedder.change(second)

	if got := text(&state.Indexing.Model); got != second.String() {
		t.Errorf("the window shows the model %q, the embedder announced %q", got, second.String())
	}
	if got := text(&state.Indexing.Recipe); got != second.Recipe() {
		t.Errorf("the window shows the recipe %q, the embedder announced %q", got, second.Recipe())
	}
}

// An embedder that announces nothing is followed by nothing, and a window without an embedder follows nothing.
func TestAnEmbedderThatAnnouncesNothingIsNotFollowed(t *testing.T) {
	followModel(nil, &State{}, nil, nil)
	followModel(silent{}, &State{}, nil, nil)
}

// silent is an embedder that cannot announce a change.
type silent struct{}

func (silent) GetModel() embeddingport.EmbeddingModel { return embeddingport.EmbeddingModel{} }

func (silent) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (silent) Close() error { return nil }

func (silent) Concurrency() int { return 1 }

func newChunking(tokens int) *announcing {
	return &announcing{model: embeddingport.EmbeddingModel{Name: "first", Dimensions: 8, Pooling: "mean", ChunkTokens: tokens}}
}

// A new chunk size cuts the notes again, so the session of the vault is built again once for it.
func TestAChangeOfTheChunkSizeBuildsTheVaultSessionAgainOnce(t *testing.T) {
	embedder := newChunking(256)
	rebuilt := 0

	followModel(embedder, &State{}, nil, func() { rebuilt++ })
	changed := embedder.model
	changed.ChunkTokens = 128
	embedder.change(changed)

	if rebuilt != 1 {
		t.Errorf("the vault session was built again %d times, want once", rebuilt)
	}
}

// The chunk size is compared with the one last seen, so a size that stays as it is rebuilds nothing.
func TestAnAnnouncementWithTheSameChunkSizeBuildsNothingAgain(t *testing.T) {
	embedder := newChunking(256)
	rebuilt := 0

	followModel(embedder, &State{}, nil, func() { rebuilt++ })
	embedder.change(embedder.model)
	other := embedder.model
	other.Name = "second"
	embedder.change(other)

	if rebuilt != 0 {
		t.Errorf("the vault session was built again %d times, want none", rebuilt)
	}
}

func TestEachChangeOfTheChunkSizeBuildsTheVaultSessionAgain(t *testing.T) {
	embedder := newChunking(256)
	rebuilt := 0

	followModel(embedder, &State{}, nil, func() { rebuilt++ })
	for _, tokens := range []int{128, 128, 256} {
		changed := embedder.model
		changed.ChunkTokens = tokens
		embedder.change(changed)
	}

	if rebuilt != 2 {
		t.Errorf("the vault session was built again %d times for 256, 128, 128, 256; want 2", rebuilt)
	}
}

// A chunk size that becomes known after the plugin was not ready is a change as well.
func TestAChunkSizeThatBecomesKnownBuildsTheVaultSessionAgain(t *testing.T) {
	embedder := newChunking(0)
	rebuilt := 0

	followModel(embedder, &State{}, nil, func() { rebuilt++ })
	changed := embedder.model
	changed.ChunkTokens = 256
	embedder.change(changed)

	if rebuilt != 1 {
		t.Errorf("the vault session was built again %d times, want once", rebuilt)
	}
}

// The model the window shows is still recorded when the chunk size changes with it.
func TestAChangeOfTheChunkSizeKeepsTheRecordedModelCurrent(t *testing.T) {
	embedder := newChunking(256)
	state := &State{}

	followModel(embedder, state, nil, func() {})
	changed := embedder.model
	changed.ChunkTokens = 128
	changed.Name = "second"
	embedder.change(changed)

	if got := text(&state.Indexing.Model); got != changed.String() {
		t.Errorf("the window shows the model %q, want %q", got, changed.String())
	}
}
