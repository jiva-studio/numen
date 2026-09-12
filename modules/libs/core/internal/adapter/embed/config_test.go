package embed_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
)

// What a vector is kept under is the model, made where the index is filled. A
// question placed elsewhere claims those rows.
func TestVectorsAreKeptUnderTheProviderThatFillsTheIndex(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Query.Use = embed.UseService

	if got := cfg.GetStoredModel().From; got != cfg.Indexing.From() {
		t.Errorf("kept under %q, filled at %q", got, cfg.Indexing.From())
	}
	if got := cfg.GetStoredModel().Name; got != cfg.Model.Name {
		t.Errorf("kept under %q, and the model is %q", got, cfg.Model.Name)
	}
}
