package embedding_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
)

func TestRecipeChangesWithTheHandlingAndNotWithMinChunkLength(t *testing.T) {
	model := embedding.Model{Name: "m", Dimensions: 4, MaxTokens: 512, Pooling: "mean", From: "p", Handling: "aaaaaaaa"}
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
	model := embedding.Model{Name: "m", Dimensions: 4, MaxTokens: 512, Pooling: "mean", From: "p", Handling: "aaaaaaaa", ChunkTokens: 256}
	smaller := model
	smaller.ChunkTokens = 128
	if smaller.Recipe() != model.Recipe() {
		t.Errorf("ChunkTokens changed the recipe: %s -> %s", model.Recipe(), smaller.Recipe())
	}
}
