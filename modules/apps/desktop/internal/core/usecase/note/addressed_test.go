package note_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// named answers what a vault holds under each name, and nothing else a query
// can be asked.
type named map[string][]string

func (n named) Named(_ context.Context, _, name string) ([]string, error) {
	return n[name], nil
}

// unreachable is a vault that cannot answer at all.
type unreachable struct{ named }

func (unreachable) Named(context.Context, string, string) ([]string, error) {
	return nil, errors.New("the index could not be read")
}

func TestALinkIsWrittenByNameWhereTheNameMeansOneNote(t *testing.T) {
	held := named{"Untitled note": {"mahabharata/Untitled note.md"}}

	to, err := note.Addressed(context.Background(), held, "v", "mahabharata/Untitled note.md")
	if err != nil {
		t.Fatalf("Addressed: %v", err)
	}

	if to.Scheme != domain.SchemeName || to.Value != "Untitled note" {
		t.Errorf("wrote %v %q, want the name alone — the folder says nothing here",
			to.Scheme, to.Value)
	}
}

// A name is read back as an exact path from the root before it is read as a
// neighbour, so the note beside the one the link is written in is reached only
// by writing where it is filed.
func TestALinkIsWrittenByPathWhereTheNameMeansAnotherNote(t *testing.T) {
	held := named{"Untitled note": {"Untitled note.md", "mahabharata/Untitled note.md"}}

	to, err := note.Addressed(context.Background(), held, "v", "mahabharata/Untitled note.md")
	if err != nil {
		t.Fatalf("Addressed: %v", err)
	}

	if to.Value != "mahabharata/Untitled note" {
		t.Errorf("wrote %q, want the path — the name alone lands on the note at the root", to.Value)
	}
	if to.Scheme != domain.SchemeName {
		t.Errorf("scheme = %v, want a name: a path from the root is one", to.Scheme)
	}
}

func TestALinkToANoteTheIndexDoesNotHoldYetIsWrittenByName(t *testing.T) {
	to, err := note.Addressed(context.Background(), named{}, "v", "mahabharata/Untitled note.md")
	if err != nil {
		t.Fatalf("Addressed: %v", err)
	}

	if to.Value != "Untitled note" {
		t.Errorf("wrote %q, want the name: nothing else is filed under it", to.Value)
	}
}

func TestALinkToNowhereIsRefused(t *testing.T) {
	if _, err := note.Addressed(context.Background(), named{}, "v", ""); err == nil {
		t.Error("Addressed() = nil error, want a refusal: a link needs a note to go to")
	}
}

func TestAVaultThatCannotBeAskedWritesNoLink(t *testing.T) {
	_, err := note.Addressed(context.Background(), unreachable{}, "v", "Note.md")

	if err == nil {
		t.Error("Addressed() = nil error, want the one the index gave: a link written " +
			"by a name nobody checked may mean another note")
	}
}
