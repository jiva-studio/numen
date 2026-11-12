package indexing_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// counting is a search source that says how many chunks it was told about, or fails with its error.
type counting struct {
	err    error
	places []searchsource.Place
}

func (*counting) Describe() searchsource.Description {
	return searchsource.Description{Name: "m", SourceRecipe: "p|m|8", Width: 8}
}

func (*counting) IndexTexts(context.Context, searchsource.Place, []searchsource.Text) ([][]float32, error) {
	return nil, nil
}

func (*counting) SearchPassages(context.Context, searchsource.Place, searchsource.Query) ([]searchsource.Match, error) {
	return nil, nil
}

func (c *counting) DescribeCoverage(_ context.Context, place searchsource.Place, counts searchsource.Counts) (string, error) {
	c.places = append(c.places, place)
	return fmt.Sprintf("%d of %d", counts.Indexed, counts.Chunks), c.err
}

// held is an index that holds twelve chunks of which three carry a vector.
type held struct{}

func (held) GetProgress(context.Context, domain.VaultID, string) (int64, int64, error) {
	return 12, 3, nil
}

func TestTheCoverageIsTheTextOfTheSourceForTheCountsOfTheIndex(t *testing.T) {
	source := &counting{}
	text, err := indexing.NewEmbedding(source, nil, held{}).DescribeCoverage(t.Context(), "01PHYSICS")
	if err != nil || text != "3 of 12" {
		t.Fatalf("coverage %q, %v; want 3 of 12", text, err)
	}
	if want := (searchsource.Place{VaultID: "01PHYSICS", Recipe: source.Describe().Recipe()}); len(source.places) != 1 || source.places[0] != want {
		t.Errorf("the source was asked for %v, want %v", source.places, want)
	}
}

func TestASourceThatDoesNotAnswerSaysNothingOfItsIndex(t *testing.T) {
	text, err := indexing.NewEmbedding(&counting{err: errors.New("the plugin is starting")}, nil, held{}).DescribeCoverage(t.Context(), "01PHYSICS")
	if err != nil || text != "" {
		t.Errorf("coverage %q, %v; want nothing", text, err)
	}
}

func TestAWindowWithoutASourceSaysNothingOfItsIndex(t *testing.T) {
	var none *indexing.Embedding
	if text, err := none.DescribeCoverage(t.Context(), "01PHYSICS"); err != nil || text != "" {
		t.Errorf("coverage %q, %v; want nothing", text, err)
	}
}
