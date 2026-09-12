package note_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// named answers what a vault holds under each name, and nothing else a query
// can be asked.
type named map[string][]string

func (n named) GetNamedPaths(_ context.Context, _ domain.VaultID, name string) ([]string, error) {
	return n[name], nil
}

// unreachable is a vault that cannot answer at all.
type unreachable struct{}

func (unreachable) GetNamedPaths(context.Context, domain.VaultID, string) ([]string, error) {
	return nil, errors.New("the index could not be read")
}

func TestALinkIsWrittenByNameWhereTheNameMeansOneNote(t *testing.T) {
	t.Parallel()
	held := named{"Untitled note": {"allotments/Untitled note.md"}}

	to, err := note.GetAddress(t.Context(), held, "v", "allotments/Untitled note.md")
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
	t.Parallel()
	held := named{"Untitled note": {"Untitled note.md", "allotments/Untitled note.md"}}

	to, err := note.GetAddress(t.Context(), held, "v", "allotments/Untitled note.md")
	if err != nil {
		t.Fatalf("Addressed: %v", err)
	}

	if to.Value != "allotments/Untitled note" {
		t.Errorf("wrote %q, want the path — the name alone lands on the note at the root", to.Value)
	}
	if to.Scheme != domain.SchemeName {
		t.Errorf("scheme = %v, want a name: a path from the root is one", to.Scheme)
	}
}

func TestALinkToANoteTheIndexDoesNotHoldYetIsWrittenByName(t *testing.T) {
	t.Parallel()
	to, err := note.GetAddress(t.Context(), named{}, "v", "allotments/Untitled note.md")
	if err != nil {
		t.Fatalf("Addressed: %v", err)
	}

	if to.Value != "Untitled note" {
		t.Errorf("wrote %q, want the name: nothing else is filed under it", to.Value)
	}
}

func TestALinkToNowhereIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := note.GetAddress(t.Context(), named{}, "v", ""); err == nil {
		t.Error("Addressed() = nil error, want a refusal: a link needs a note to go to")
	}
}

func TestAVaultThatCannotBeAskedWritesNoLink(t *testing.T) {
	t.Parallel()
	_, err := note.GetAddress(t.Context(), unreachable{}, "v", "Note.md")

	if err == nil {
		t.Error("Addressed() = nil error, want the one the index gave: a link written " +
			"by a name nobody checked may mean another note")
	}
}

// A note whose name a link is read up to is reached by no link.
//
// A link runs to the first `#` or `|`, and what stands after it names a heading
// or the words to show. So a link written to such a name reads back as the name
// in front of it, which is another note or none, and rewriting the entry from
// what was read moves the link there for good.
func TestANoteNoLinkReachesIsRefused(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"study/Study #1.md",
		"study/Verbs | strong.md",
		"study/#1.md",
	} {
		held := named{domain.Basename(path): {path}}

		to, err := note.GetAddress(t.Context(), held, "v", path)
		if !errors.Is(err, note.ErrUnaddressable) {
			t.Errorf("%s is addressed as %v %q, and answered %v",
				path, to.Scheme, to.Value, err)
		}
	}
}
