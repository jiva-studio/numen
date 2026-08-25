package note_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
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

// drawnOn renders the line between two notes as it is seen from one of them:
// the seat, the label where there is one, and whether the edge is mutual.
func drawnOn(t *testing.T, files map[string]string, from, to string) string {
	t.Helper()
	for _, r := range neighbourhoodOf(t, files, from).Related {
		if r.Path != to {
			continue
		}
		drawn := []string{string(r.Seat)}
		if r.Label != "" {
			drawn = append(drawn, r.Label)
		}
		if r.Mutual {
			drawn = append(drawn, "mutual")
		}
		return strings.Join(drawn, " ")
	}
	return "nothing"
}

// TestALabelWrittenAtEitherEndIsDrawn.
//
// Either end of a relationship can name it, and both ends naming it makes the
// edge mutual. The word shown is then the one the note in focus wrote, so the
// two notes read the same edge in their own language.
func TestALabelWrittenAtEitherEndIsDrawn(t *testing.T) {
	written := func(title, links string) string {
		if links == "" {
			return "---\ntitle: " + title + "\n---\n\n# " + title + "\n"
		}
		return "---\ntitle: " + title + "\nlinks:\n" + links + "---\n\n# " + title + "\n"
	}

	for _, c := range []struct {
		name               string
		area, idea         string
		fromArea, fromIdea string
	}{
		{
			name:     "the far end alone names it",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n",
			idea:     "  - to: \"[[Area]]\"\n    role: child\n    label: Жлоб\n",
			fromArea: "parent Жлоб mutual",
			fromIdea: "child Жлоб mutual",
		},
		{
			name:     "each end has its own word for it",
			area:     "  - to: \"[[Idea]]\"\n    role: jump\n    label: внучатый племянник\n",
			idea:     "  - to: \"[[Area]]\"\n    role: jump\n    label: питамаха, дед рода\n",
			fromArea: "jump внучатый племянник mutual",
			fromIdea: "jump питамаха, дед рода mutual",
		},
		{
			name:     "the near end alone names it",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n    label: дом\n",
			idea:     "  - to: \"[[Area]]\"\n    role: child\n",
			fromArea: "parent дом mutual",
			fromIdea: "child дом mutual",
		},
		{
			name:     "written at one end only",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n    label: дом\n",
			idea:     "",
			fromArea: "parent дом",
			fromIdea: "child дом",
		},
		{
			name:     "mutual and named by neither",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n",
			idea:     "  - to: \"[[Area]]\"\n    role: child\n",
			fromArea: "parent mutual",
			fromIdea: "child mutual",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			files := map[string]string{
				"Area.md": written("Area", c.area),
				"Idea.md": written("Idea", c.idea),
			}
			if got := drawnOn(t, files, "Area.md", "Idea.md"); got != c.fromArea {
				t.Errorf("from Area: got %q, want %q", got, c.fromArea)
			}
			if got := drawnOn(t, files, "Idea.md", "Area.md"); got != c.fromIdea {
				t.Errorf("from Idea: got %q, want %q", got, c.fromIdea)
			}
		})
	}
}

// TestEndsThatDisagreeAnswerNothing. A pair who are each other's parent name
// two relationships, and the one drawn is the one whose seat wins.
func TestEndsThatDisagreeAnswerNothing(t *testing.T) {
	mutualParents := map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: parent\n    label: несушка\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: parent\n    label: из яйца\n---\n\n# Egg\n",
	}
	if got := drawnOn(t, mutualParents, "Chicken.md", "Egg.md"); got != "parent несушка" {
		t.Errorf("got %q, want the parent seat and the word written here", got)
	}
	if got := drawnOn(t, mutualParents, "Egg.md", "Chicken.md"); got != "parent из яйца" {
		t.Errorf("got %q, want the parent seat and the word written here", got)
	}

	// Each calls the other its child, so the parent seat is earned from the far
	// end and carries the word written there.
	mutualChildren := map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: child\n    label: снесённое\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: child\n    label: вылупившийся\n---\n\n# Egg\n",
	}
	if got := drawnOn(t, mutualChildren, "Chicken.md", "Egg.md"); got != "parent вылупившийся" {
		t.Errorf("got %q, want the parent seat and the word written at the end it came from", got)
	}
	if got := drawnOn(t, mutualChildren, "Egg.md", "Chicken.md"); got != "parent снесённое" {
		t.Errorf("got %q, want the parent seat and the word written at the end it came from", got)
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

// TestTheHigherSeatWins is the other half of taking one seat: which one.
//
// Written from the mirrored end, so the child arrives first and the parent
// second. Keeping whichever was seen first would seat it as a child; the rule
// is that a parent outranks one.
func TestTheHigherSeatWins(t *testing.T) {
	n := neighbourhoodOf(t, map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: child\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: child\n---\n\n# Egg\n",
	}, "Chicken.md")

	if got := seatsOf(n); !slices.Equal(got, []string{"parent Egg.md"}) {
		t.Errorf("got %v, want parent — it outranks the child seat the same pair also earns", got)
	}
}

// TestANeighbourhoodStaysInsideItsVault. A link by identifier resolves in
// whichever connected vault holds the note, so a neighbourhood that does not
// check where one landed draws a note from somewhere else — or, when the two
// vaults file a note at the same path, the wrong note under the right name.
func TestANeighbourhoodStaysInsideItsVault(t *testing.T) {
	const shared = "notes/Entropy.md"
	db, here := indexed(t, map[string]string{
		shared:    "---\ntitle: Entropy here\nid: 01M02ACGM0FYMSXNDP29C90JN1\n---\n\n# Entropy here\n",
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JN2\"\n    role: child\n---\n\n# Area\n",
	})
	addVault(t, db, testsupport.NewVault(t, map[string]string{
		shared: "---\ntitle: Entropy elsewhere\nid: 01M02ACGM0FYMSXNDP29C90JN2\n---\n\n# Entropy elsewhere\n",
	}))

	n, err := note.ShowNeighbourhood{Links: db.Links(), Notes: db.Queries()}.
		Execute(t.Context(), here, "Area.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := seatsOf(n); len(got) != 0 {
		t.Errorf("got %v — the note that identifier names is in another vault", got)
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
//
// Every note is called the same thing, on purpose. A picture ordered by what a
// note is called is ordered by nothing at all when they share a name, and the
// vault where that is true is the one a person notices it in: twelve daily
// notes, twelve chapters, twelve "Notes".
func TestTheSameVaultDrawsTheSameWayTwice(t *testing.T) {
	files := map[string]string{"Area.md": "---\ntitle: Area\n---\n\n# Area\n"}
	for i := range 12 {
		files[fmt.Sprintf("Child%02d.md", i)] = fmt.Sprintf(
			"---\ntitle: Notes\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Notes %02d\n", i)
	}

	first := seatsOf(neighbourhoodOf(t, files, "Area.md"))
	for range 8 {
		if again := seatsOf(neighbourhoodOf(t, files, "Area.md")); !slices.Equal(first, again) {
			t.Fatalf("two runs disagree:\n  %v\n  %v", first, again)
		}
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
