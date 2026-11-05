package editor

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
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

	followModel(embedder, state, nil)
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
	followModel(nil, &State{}, nil)
	followModel(silent{}, &State{}, nil)
}

// silent is an embedder that cannot announce a change.
type silent struct{}

func (silent) GetModel() embeddingport.EmbeddingModel { return embeddingport.EmbeddingModel{} }

func (silent) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (silent) Close() error { return nil }

func (silent) Concurrency() int { return 1 }
