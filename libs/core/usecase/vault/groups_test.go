package vault

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// TestFlushClearsTheBackingArray: a flushed group must not keep referencing the
// notes it wrote through the slice's backing array, so the GC can
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

// newRecordingGroup is a group whose writes are the sizes of what each one held.
func newRecordingGroup() (*grouping, *[]int) {
	var writes []int
	return &grouping{write: func(_ context.Context, notes []domain.Note) error {
		writes = append(writes, len(notes))
		return nil
	}}, &writes
}

// A group is written when it holds five hundred notes.
func TestAGroupIsSavedAtFiveHundredNotes(t *testing.T) {
	t.Parallel()
	g, writes := newRecordingGroup()

	for range 499 {
		if err := g.add(t.Context(), domain.Note{}, 1); err != nil {
			t.Fatal(err)
		}
	}
	if len(*writes) != 0 {
		t.Fatalf("a group of 499 notes was written: %v", *writes)
	}
	if err := g.add(t.Context(), domain.Note{}, 1); err != nil {
		t.Fatal(err)
	}
	if len(*writes) != 1 || (*writes)[0] != 500 {
		t.Errorf("the writes are %v, want one of 500", *writes)
	}
}

// A group is written when the files it holds reach eight mebibytes, and each
// group after a write starts from nothing.
func TestAGroupIsSavedAtEightMebibytes(t *testing.T) {
	t.Parallel()
	const eightMiB = 8 << 20
	g, writes := newRecordingGroup()

	for round := range 2 {
		if err := g.add(t.Context(), domain.Note{}, eightMiB-1); err != nil {
			t.Fatal(err)
		}
		if len(*writes) != round {
			t.Fatalf("round %d: a group one byte short of the bound was written: %v", round, *writes)
		}
		if err := g.add(t.Context(), domain.Note{}, 1); err != nil {
			t.Fatal(err)
		}
		if len(*writes) != round+1 {
			t.Fatalf("round %d: a group at the bound was not written: %v", round, *writes)
		}
	}
	if (*writes)[0] != 2 || (*writes)[1] != 2 {
		t.Errorf("the groups held %v notes, want two each", *writes)
	}
}

// Flushing a group that holds nothing writes nothing.
func TestFlushingAnEmptyGroupWritesNothing(t *testing.T) {
	t.Parallel()
	g, writes := newRecordingGroup()

	if err := g.flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(*writes) != 0 {
		t.Errorf("an empty group was written: %v", *writes)
	}
}
