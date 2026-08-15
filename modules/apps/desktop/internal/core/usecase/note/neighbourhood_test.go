package note_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// seatsOf renders a neighbourhood as `seat path` lines, which is what these
// tests are about: who is shown, and where.
func seatsOf(n domain.Neighbourhood) []string {
	out := make([]string, 0, len(n.Related))
	for _, r := range n.Related {
		out = append(out, string(r.Seat)+" "+r.Path)
	}
	return out
}

func neighbourhoodOf(t *testing.T, files map[string]string, path string) domain.Neighbourhood {
	t.Helper()
	db, v := indexed(t, files)
	n, err := note.ShowNeighbourhood{Links: db.Links(), Notes: db.Queries()}.
		Execute(t.Context(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestBothEndsOfAnEdgeAreOneRelationship. `parent: B` in A and `child: A` in B
// say the same thing, so A must be B's child either way it was written.
func TestBothEndsOfAnEdgeAreOneRelationship(t *testing.T) {
	written := neighbourhoodOf(t, map[string]string{
		"Area.md":  "---\ntitle: Area\n---\n\n# Area\n",
		"Idea.md":  "---\ntitle: Idea\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Idea\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n",
	}, "Area.md")
	if got := seatsOf(written); !slices.Equal(got, []string{"child Idea.md"}) {
		t.Errorf("a note that names Area its parent is not Area's child: %v", got)
	}

	reversed := neighbourhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"[[Idea]]\"\n    role: child\n---\n\n# Area\n",
		"Idea.md": "---\ntitle: Idea\n---\n\n# Idea\n",
	}, "Area.md")
	if got := seatsOf(reversed); !slices.Equal(got, []string{"child Idea.md"}) {
		t.Errorf("the same edge written from the other end: %v", got)
	}
}

// TestSiblingsAreTheOtherChildrenOfAParent. Nothing stores a sibling; it is
// what the shared parent knows.
func TestSiblingsAreTheOtherChildrenOfAParent(t *testing.T) {
	n := neighbourhoodOf(t, map[string]string{
		"Area.md":  "---\ntitle: Area\n---\n\n# Area\n",
		"One.md":   "---\ntitle: One\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# One\n",
		"Two.md":   "---\ntitle: Two\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Two\n",
		"Three.md": "---\ntitle: Three\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Three\n",
	}, "One.md")

	want := []string{"parent Area.md", "sibling Three.md", "sibling Two.md"}
	if got := seatsOf(n); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestANoteTakesOneSeat. A pair that are each other's parent is a contradiction
// the vault is allowed to contain, and a picture that draws one note twice is
// not a picture.
func TestANoteTakesOneSeat(t *testing.T) {
	n := neighbourhoodOf(t, map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: parent\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: parent\n---\n\n# Egg\n",
	}, "Chicken.md")

	if got := seatsOf(n); !slices.Equal(got, []string{"parent Egg.md"}) {
		t.Errorf("got %v, want the first seat it qualifies for and no second one", got)
	}
}

// TestOnlyNavigableLinksTakeASeat. A wikilink in prose and an attachment are
// links, and neither is a place in the hierarchy.
func TestOnlyNavigableLinksTakeASeat(t *testing.T) {
	n := neighbourhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"https://example.org/paper\"\n    role: attachment\n" +
			"  - to: \"[[Idea]]\"\n    role: jump\n---\n\n# Area\n\nMentioned in prose: [[Other]].\n",
		"Idea.md":  "---\ntitle: Idea\n---\n\n# Idea\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n",
	}, "Area.md")

	if got := seatsOf(n); !slices.Equal(got, []string{"jump Idea.md"}) {
		t.Errorf("got %v — prose links and attachments are not seats", got)
	}
}

// TestADanglingLinkHasNoSeat: there is nothing to draw and nowhere to go.
func TestADanglingLinkHasNoSeat(t *testing.T) {
	n := neighbourhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"[[Nothing by that name]]\"\n    role: child\n---\n\n# Area\n",
	}, "Area.md")

	if got := seatsOf(n); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
	if n.Focus.Title != "Area" {
		t.Errorf("focus = %+v", n.Focus)
	}
}

// TestTheSameVaultDrawsTheSameWayTwice.
func TestTheSameVaultDrawsTheSameWayTwice(t *testing.T) {
	files := map[string]string{"Area.md": "---\ntitle: Area\n---\n\n# Area\n"}
	for i := range 12 {
		files[fmt.Sprintf("Child%02d.md", i)] = fmt.Sprintf(
			"---\ntitle: Child %02d\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Child %02d\n", i, i)
	}

	first := seatsOf(neighbourhoodOf(t, files, "Area.md"))
	second := seatsOf(neighbourhoodOf(t, files, "Area.md"))
	if !slices.Equal(first, second) {
		t.Errorf("two runs disagree:\n  %v\n  %v", first, second)
	}
	if len(first) != 12 {
		t.Errorf("%d children, want 12", len(first))
	}
}

// TestAVaultOpensOnItsFirstNote.
func TestAVaultOpensOnItsFirstNote(t *testing.T) {
	db, v := indexed(t, map[string]string{
		"Area.md": "---\ntitle: Area\n---\n\n# Area\n",
		"Idea.md": "---\ntitle: Idea\n---\n\n# Idea\n",
	})

	opening, found, err := db.Queries().Opening(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("a vault with two notes opens on neither")
	}
	if opening.Title == "" || opening.Path == "" {
		t.Errorf("opening = %+v", opening)
	}

	again, _, err := db.Queries().Opening(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again != opening {
		t.Errorf("a vault opens somewhere different each time: %+v then %+v", opening, again)
	}
}

func TestAnEmptyVaultOpensOnNothing(t *testing.T) {
	db, v := indexed(t, map[string]string{})

	_, found, err := db.Queries().Opening(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("an empty vault opened on something")
	}
}
