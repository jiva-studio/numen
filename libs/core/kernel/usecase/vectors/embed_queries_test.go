package vectors

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
)

// documentsOnly has one kind of vector.
type documentsOnly struct{ calls []string }

func (*documentsOnly) GetModel() embedding.Model { return embedding.Model{} }
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
	if _, err := NewEmbedQueries(e, e).Execute(t.Context(), []string{"q"}); err != nil {
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
	if _, err := NewEmbedQueries(e, nil).Execute(t.Context(), []string{"q"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(e.calls, ","); got != "document" {
		t.Errorf("calls for %q, want document", got)
	}
}
