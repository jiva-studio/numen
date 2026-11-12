package vectors

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
)

// servedBy is an embedder whose model names the connection it came from: the type of the connection
// with its address.
type servedBy struct {
	*embedder
	from string
}

func (s servedBy) Describe() searchsource.Description {
	model := s.embedder.Describe()
	model.SourceRecipe = s.from + "|" + model.SourceRecipe
	return model
}

// A model of another width at another address is another recipe: every chunk is embedded again, the
// vectors of the old recipe stay stored beside the new ones until a rebuild takes them out, and the
// index is fitted to the new width before any text is embedded.
func TestEmbeddingWithAnotherWidthAndAddressWritesNewVectorsAndKeepsTheOldOnes(t *testing.T) {
	ctx := t.Context()
	index, shelf := indexfake.NewStore(), indexfake.NewLibrary()
	shelf.Put(bookPath, domain.KindFile, indexfake.CreateBook(t, "A Book", indexfake.GetWords(indexfake.GetSanskrit(), 900)), 1)
	small := cutBooks(t, index, shelf, indexfake.GetFirst())

	narrow := servedBy{&embedder{dims: 8}, "md.numen.connections.llm/openai https://api.openai.com/v1"}
	large := servedBy{&embedder{dims: 16}, "md.numen.connections.llm/openai-compatible http://127.0.0.1:11434/v1"}
	if narrow.Describe().Recipe() == large.Describe().Recipe() {
		t.Fatal("two widths at two addresses share one recipe")
	}

	embed := newEmbed(indexfake.Vaults{indexfake.GetFirst().ID: shelf}, index, index)
	embed.BatchCharacters = 4000
	embed.Source = sourcefake.Wrap(narrow)
	if _, err := embed.Execute(ctx, indexfake.GetFirst()); err != nil {
		t.Fatal(err)
	}

	fit := &vectorIndex{}
	fitter := NewFitIndex(fit, nil)
	fitter.ShouldRebuildIndex = true
	if err := fitter.Execute(ctx, large); err != nil {
		t.Fatal(err)
	}
	if fit.dims != 16 || fit.recipe != large.Describe().Recipe() {
		t.Errorf("the index was fitted to %d dimensions under %q, want 16 under the recipe of the new address", fit.dims, fit.recipe)
	}
	if fit.calls[len(fit.calls)-2] != "delete "+large.Describe().Recipe() {
		t.Errorf("calls = %v, want the vectors of every other recipe taken out on a rebuild", fit.calls)
	}

	embed.Source = sourcefake.Wrap(large)
	res, err := embed.Execute(ctx, indexfake.GetFirst())
	if err != nil {
		t.Fatal(err)
	}
	if res.Embedded != len(small) || res.Reused != 0 {
		t.Errorf("the second run embedded %d and reused %d of %d chunks, want all embedded again", res.Embedded, res.Reused, len(small))
	}
	for _, chunk := range small {
		stored := index.GetVectors(chunk)
		if len(stored) != 2 || stored[0].Recipe != narrow.Describe().Recipe() || stored[1].Recipe != large.Describe().Recipe() {
			t.Fatalf("chunk %d holds %d vectors, want the old one and the new one", chunk.ID, len(stored))
		}
	}
}
