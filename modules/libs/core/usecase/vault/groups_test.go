package vault

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// TestFlushClearsTheBackingArray: a flushed group must not go on holding the
// notes it just wrote through the slice's backing array, so the GC can
// reclaim them before the next group fills those slots.
func TestFlushClearsTheBackingArray(t *testing.T) {
	g := &grouping{write: func(context.Context, []domain.Note) error { return nil }}
	note := domain.Note{
		Title:       "kept",
		Frontmatter: map[string]any{"k": "v"},
		Body:        "body",
	}
	if err := g.add(t.Context(), note, 1); err != nil {
		t.Fatal(err)
	}
	if err := g.flush(t.Context()); err != nil {
		t.Fatal(err)
	}

	full := g.notes[:cap(g.notes)]
	if full[0].Title != "" || full[0].Frontmatter != nil || full[0].Body != "" {
		t.Errorf("flush left a reference in the backing array: %+v", full[0])
	}
}
