package vectors

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
)

// queriesOnly makes the vectors of queries and counts them.
type queriesOnly struct{ asked int }

func (q *queriesOnly) EmbedQueries(_ context.Context, texts []string) ([][]float32, error) {
	q.asked += len(texts)
	return make([][]float32, len(texts)), nil
}

func describeSource() searchsource.Description {
	return searchsource.Description{Name: "m", SourceRecipe: "p|m|4", Width: 4}
}

func TestEmbedTextsAsksTheQueriesOfTheSourceAndNamesItsRecipe(t *testing.T) {
	queries := &queriesOnly{}
	got, err := NewEmbedQueries(queries).EmbedTexts(t.Context(), describeSource, []string{"entropy", "heat"})
	if err != nil {
		t.Fatal(err)
	}
	if queries.asked != 2 || len(got.Vectors) != 2 || got.Recipe != describeSource().Recipe() {
		t.Errorf("asked %d queries for %+v", queries.asked, got)
	}
}

func TestEmbedTextsRefusesMoreTextsThanOneCallEmbeds(t *testing.T) {
	texts := make([]string, plugin.MaxEmbeddedTexts+1)
	if _, err := NewEmbedQueries(&queriesOnly{}).EmbedTexts(t.Context(), describeSource, texts); !errors.Is(err, plugin.ErrLimitExceeded) {
		t.Errorf("err = %v, want ErrLimitExceeded", err)
	}
}
