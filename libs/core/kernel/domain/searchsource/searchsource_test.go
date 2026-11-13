package searchsource_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
)

func TestRecipeChangesWithTheSourceRecipeAndNotWithTheChunkSizes(t *testing.T) {
	source := searchsource.Description{Name: "m", SourceRecipe: "p|m|4|aaaaaaaa", Width: 4}
	base := source.Recipe()

	changed := source
	changed.SourceRecipe = "p|m|4|bbbbbbbb"
	if changed.Recipe() == base {
		t.Error("another source recipe left the recipe as it was")
	}
	sized := source
	sized.MinTextLength, sized.ChunkTokens = 400, 128
	if sized.Recipe() != base {
		t.Errorf("the chunk sizes changed the recipe: %s", sized.Recipe())
	}
	if !strings.HasPrefix(base, "p|m|4|aaaaaaaa|int8@") {
		t.Errorf("recipe %q does not start with the source recipe and the quantization", base)
	}
}
