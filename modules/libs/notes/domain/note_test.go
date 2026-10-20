package domain_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestEveryNoteTypeIsKnownAndNothingElseIs(t *testing.T) {
	want := []domain.NoteType{domain.TypeNote, domain.TypeDeck, domain.TypeStencil, domain.TypePreset}
	if got := domain.NoteTypes(); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, one := range want {
		if !domain.IsKnownNoteType(one) {
			t.Errorf("%q is not known", one)
		}
	}
	for _, one := range []domain.NoteType{"", "Note", "folder", "decks"} {
		if domain.IsKnownNoteType(one) {
			t.Errorf("%q is known", one)
		}
	}
}
