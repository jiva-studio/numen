package embeddingport_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

// documentsOnly has one kind of vector.
type documentsOnly struct{ calls []string }

func (*documentsOnly) GetModel() embeddingport.EmbeddingModel { return embeddingport.EmbeddingModel{} }
func (d *documentsOnly) Embed(_ context.Context, texts []string) ([][]float32, error) {
	d.calls = append(d.calls, "document")
	return make([][]float32, len(texts)), nil
}
func (*documentsOnly) Concurrency() int { return 1 }
func (*documentsOnly) Close() error     { return nil }

// twoKinds tells a query from a document.
type twoKinds struct{ documentsOnly }

func (d *twoKinds) EmbedQueries(_ context.Context, texts []string) ([][]float32, error) {
	d.calls = append(d.calls, "query")
	return make([][]float32, len(texts)), nil
}

func TestEmbedQueriesAsksAnEmbedderThatTellsQueriesApartForQueryVectors(t *testing.T) {
	e := &twoKinds{}
	if _, err := embeddingport.EmbedQueries(t.Context(), e, []string{"q"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Embed(t.Context(), []string{"d"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(e.calls, ","); got != "query,document" {
		t.Errorf("calls for %q, want query then document", got)
	}
}

func TestEmbedQueriesAsksAnyOtherEmbedderForItsOnlyKind(t *testing.T) {
	e := &documentsOnly{}
	if _, err := embeddingport.EmbedQueries(t.Context(), e, []string{"q"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(e.calls, ","); got != "document" {
		t.Errorf("calls for %q, want document", got)
	}
}

func TestRecipeChangesWithTheHandlingAndNotWithMinChunkLength(t *testing.T) {
	model := embeddingport.EmbeddingModel{Name: "m", Dimensions: 4, MaxTokens: 512, Pooling: "mean", From: "p", Handling: "aaaaaaaa"}
	base := model.Recipe()

	changed := model
	changed.Handling = "bbbbbbbb"
	if changed.Recipe() == base {
		t.Error("another handling left the recipe as it was")
	}
	short := model
	short.MinChunkLength = 400
	if short.Recipe() != base {
		t.Errorf("MinChunkLength changed the recipe: %s", short.Recipe())
	}
	if !strings.Contains(base, "|mean|aaaaaaaa|") {
		t.Errorf("recipe %q does not hold the handling after the pooling", base)
	}
}

func TestRecipeDoesNotChangeWithTheChunkTokens(t *testing.T) {
	model := embeddingport.EmbeddingModel{Name: "m", Dimensions: 4, MaxTokens: 512, Pooling: "mean", From: "p", Handling: "aaaaaaaa", ChunkTokens: 256}
	smaller := model
	smaller.ChunkTokens = 128
	if smaller.Recipe() != model.Recipe() {
		t.Errorf("ChunkTokens changed the recipe: %s -> %s", model.Recipe(), smaller.Recipe())
	}
}
