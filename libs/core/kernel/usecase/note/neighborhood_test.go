package note_test

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func relationsOf(n domain.Neighborhood) []string {
	out := make([]string, 0, len(n.Related))
	for _, r := range n.Related {
		out = append(out, string(r.Relation)+" "+r.Path)
	}
	return out
}

func neighborhoodOf(t *testing.T, files map[string]string, path string) domain.Neighborhood {
	t.Helper()
	db, v := newIndexedVault(t, files)
	n, err := query.NewShowNeighborhood(db.NoteQueries(), db.NoteQueries()).
		Execute(t.Context(), v, path)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestBothEndsOfAnEdgeAreOneRelationship. `parent: B` in A and `child: A` in B
// say the same thing, so A must be B's child either way it was written.
func TestBothEndsOfAnEdgeAreOneRelationship(t *testing.T) {
	t.Parallel()
	content := neighborhoodOf(t, map[string]string{
		"Area.md":  "---\ntitle: Area\n---\n\n# Area\n",
		"Idea.md":  "---\ntitle: Idea\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Idea\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n",
	}, "Area.md")
	if got := relationsOf(content); !slices.Equal(got, []string{"child Idea.md"}) {
		t.Errorf("a note that names Area its parent is not Area's child: %v", got)
	}

	reversed := neighborhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"[[Idea]]\"\n    role: child\n---\n\n# Area\n",
		"Idea.md": "---\ntitle: Idea\n---\n\n# Idea\n",
	}, "Area.md")
	if got := relationsOf(reversed); !slices.Equal(got, []string{"child Idea.md"}) {
		t.Errorf("the same edge written from the other end: %v", got)
	}
}

func describeSide(s *domain.Side) string {
	if s == nil {
		return "none"
	}
	text := strconv.Quote(s.Label)
	if s.Marker == domain.MarkerArrow {
		text += " arrow"
	}
	return text
}

func describeLine(c domain.Connection) string {
	return c.A + ">" + c.B + " ab=" + describeSide(c.AB) + " ba=" + describeSide(c.BA)
}

func linesOf(n domain.Neighborhood) []string {
	out := make([]string, 0, len(n.Connections))
	for _, c := range n.Connections {
		out = append(out, describeLine(c))
	}
	return out
}

// TestEachEndOfALineKeepsItsOwnRecord. Either end of a relationship can name it,
// and the line carries the record each note wrote, whichever note is in focus.
func TestEachEndOfALineKeepsItsOwnRecord(t *testing.T) {
	t.Parallel()
	content := func(title, links string) string {
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
			idea:     "  - to: \"[[Area]]\"\n    role: child\n    label: the top plot\n",
			fromArea: `Idea.md>Area.md ab="the top plot" ba=""`,
			fromIdea: `Idea.md>Area.md ab="the top plot" ba=""`,
		},
		{
			name:     "each end has its own word for it",
			area:     "  - to: \"[[Idea]]\"\n    role: jump\n    label: the newest tenant\n",
			idea:     "  - to: \"[[Area]]\"\n    role: jump\n    label: the oldest tenant\n",
			fromArea: `Idea.md>Area.md ab="the oldest tenant" ba="the newest tenant"`,
			fromIdea: `Area.md>Idea.md ab="the newest tenant" ba="the oldest tenant"`,
		},
		{
			name:     "the near end alone names it",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n    label: the site\n",
			idea:     "  - to: \"[[Area]]\"\n    role: child\n",
			fromArea: `Idea.md>Area.md ab="" ba="the site"`,
			fromIdea: `Idea.md>Area.md ab="" ba="the site"`,
		},
		{
			name:     "written at one end only",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n    label: the site\n",
			idea:     "",
			fromArea: `Idea.md>Area.md ab=none ba="the site"`,
			fromIdea: `Idea.md>Area.md ab=none ba="the site"`,
		},
		{
			name:     "an arrow at the end the record points to",
			area:     "  - to: \"[[Idea]]\"\n    role: child\n    marker: arrow\n",
			idea:     "",
			fromArea: `Area.md>Idea.md ab="" arrow ba=none`,
			fromIdea: `Area.md>Idea.md ab="" arrow ba=none`,
		},
		{
			name:     "named by neither",
			area:     "  - to: \"[[Idea]]\"\n    role: parent\n",
			idea:     "  - to: \"[[Area]]\"\n    role: child\n",
			fromArea: `Idea.md>Area.md ab="" ba=""`,
			fromIdea: `Idea.md>Area.md ab="" ba=""`,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			files := map[string]string{
				"Area.md": content("Area", c.area),
				"Idea.md": content("Idea", c.idea),
			}
			if got := linesOf(neighborhoodOf(t, files, "Area.md")); !slices.Equal(got, []string{c.fromArea}) {
				t.Errorf("from Area: got %q, want %q", got, c.fromArea)
			}
			if got := linesOf(neighborhoodOf(t, files, "Idea.md")); !slices.Equal(got, []string{c.fromIdea}) {
				t.Errorf("from Idea: got %q, want %q", got, c.fromIdea)
			}
		})
	}
}

// TestEndsThatDisagreeKeepOnlyTheRecordsOfTheWinningDirection. A pair who are each
// other's parent name two relationships, and the line drawn is the one whose
// relation wins, with the records that say the same.
func TestEndsThatDisagreeKeepOnlyTheRecordsOfTheWinningDirection(t *testing.T) {
	t.Parallel()
	mutualParents := map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: parent\n    label: lays them\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: parent\n    label: came out of one\n---\n\n# Egg\n",
	}
	if got := linesOf(neighborhoodOf(t, mutualParents, "Chicken.md")); !slices.Equal(got, []string{`Egg.md>Chicken.md ab=none ba="lays them"`}) {
		t.Errorf("got %q, want the neighbor as the parent and the word written here", got)
	}
	if got := linesOf(neighborhoodOf(t, mutualParents, "Egg.md")); !slices.Equal(got, []string{`Chicken.md>Egg.md ab=none ba="came out of one"`}) {
		t.Errorf("got %q, want the neighbor as the parent and the word written here", got)
	}

	// Each calls the other its child, so the parent relation is earned from the far
	// end and carries the word written there.
	mutualChildren := map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: child\n    label: was laid\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: child\n    label: hatched\n---\n\n# Egg\n",
	}
	if got := linesOf(neighborhoodOf(t, mutualChildren, "Chicken.md")); !slices.Equal(got, []string{`Egg.md>Chicken.md ab="hatched" ba=none`}) {
		t.Errorf("got %q, want the word written at the end it came from", got)
	}
	if got := linesOf(neighborhoodOf(t, mutualChildren, "Egg.md")); !slices.Equal(got, []string{`Chicken.md>Egg.md ab="was laid" ba=none`}) {
		t.Errorf("got %q, want the word written at the end it came from", got)
	}
}

// TestLinesJoinNotesThatAreNotTheFocus. Every pair of drawn notes that a record
// joins is a line with its records, in each direction a record is written.
func TestLinesJoinNotesThatAreNotTheFocus(t *testing.T) {
	t.Parallel()
	const area = "---\ntitle: Area\nlinks:\n  - to: \"[[Idea]]\"\n    role: child\n  - to: \"[[Plan]]\"\n    role: child\n---\n\n# Area\n"
	note := func(title, links string) string {
		if links == "" {
			return "---\ntitle: " + title + "\n---\n\n# " + title + "\n"
		}
		return "---\ntitle: " + title + "\nlinks:\n" + links + "---\n\n# " + title + "\n"
	}
	const ideaJump = "  - to: \"[[Plan]]\"\n    role: jump\n    label: feeds\n    marker: arrow\n"
	const planJump = "  - to: \"[[Idea]]\"\n    role: jump\n    label: needs\n"

	for _, c := range []struct {
		name, idea, plan string
		want             string
	}{
		{"the earlier note writes", ideaJump, "", `Plan.md>Idea.md ab=none ba="feeds" arrow`},
		{"the later note writes", "", planJump, `Plan.md>Idea.md ab="needs" ba=none`},
		{"both write", ideaJump, planJump, `Plan.md>Idea.md ab="needs" ba="feeds" arrow`},
		{"neither writes", "", "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			n := neighborhoodOf(t, map[string]string{
				"Area.md": area, "Idea.md": note("Idea", c.idea), "Plan.md": note("Plan", c.plan),
			}, "Area.md")
			want := []string{"Area.md>Idea.md ab=\"\" ba=none", "Area.md>Plan.md ab=\"\" ba=none"}
			if c.want != "" {
				want = append(want, c.want)
			}
			if got := linesOf(n); !slices.Equal(got, want) {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// TestLinesJoinSiblingsToEachOtherAndToTheirParent. A sibling is not joined to
// the focus, yet the records it writes are lines like any other.
func TestLinesJoinSiblingsToEachOtherAndToTheirParent(t *testing.T) {
	t.Parallel()
	n := neighborhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\n---\n\n# Area\n",
		"One.md":  "---\ntitle: One\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# One\n",
		"Two.md": "---\ntitle: Two\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n    label: part of\n    marker: arrow\n" +
			"  - to: \"[[Three]]\"\n    role: jump\n    label: before\n    marker: arrow\n---\n\n# Two\n",
		"Three.md": "---\ntitle: Three\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Three\n",
	}, "One.md")

	want := []string{
		`Area.md>One.md ab=none ba=""`,
		`Area.md>Three.md ab=none ba=""`,
		`Area.md>Two.md ab=none ba="part of" arrow`,
		`Two.md>Three.md ab="before" arrow ba=none`,
	}
	if got := linesOf(n); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestSiblingsAreTheOtherChildrenOfAParent. Nothing stores a sibling; it is
// derived from the shared parent.
func TestSiblingsAreTheOtherChildrenOfAParent(t *testing.T) {
	t.Parallel()
	n := neighborhoodOf(t, map[string]string{
		"Area.md":  "---\ntitle: Area\n---\n\n# Area\n",
		"One.md":   "---\ntitle: One\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# One\n",
		"Two.md":   "---\ntitle: Two\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Two\n",
		"Three.md": "---\ntitle: Three\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Three\n",
	}, "One.md")

	want := []string{"parent Area.md", "sibling Three.md", "sibling Two.md"}
	if got := relationsOf(n); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestANoteTakesOneRelation. A pair that are each other's parent is a contradiction
// the vault is allowed to contain, and a picture that draws one note twice is
// not a picture.
func TestANoteConsumesOneRelation(t *testing.T) {
	t.Parallel()
	n := neighborhoodOf(t, map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: parent\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: parent\n---\n\n# Egg\n",
	}, "Chicken.md")

	if got := relationsOf(n); !slices.Equal(got, []string{"parent Egg.md"}) {
		t.Errorf("got %v, want the first relation it qualifies for and no second one", got)
	}
}

// TestTheHigherRelationWins is the other half of taking one relation: which one.
//
// Written from the mirrored end, so the child arrives first and the parent
// second. Keeping whichever was seen first would make it a child; the rule
// is that a parent outranks one.
func TestTheHigherRelationWins(t *testing.T) {
	t.Parallel()
	n := neighborhoodOf(t, map[string]string{
		"Chicken.md": "---\ntitle: Chicken\nlinks:\n  - to: \"[[Egg]]\"\n    role: child\n---\n\n# Chicken\n",
		"Egg.md":     "---\ntitle: Egg\nlinks:\n  - to: \"[[Chicken]]\"\n    role: child\n---\n\n# Egg\n",
	}, "Chicken.md")

	if got := relationsOf(n); !slices.Equal(got, []string{"parent Egg.md"}) {
		t.Errorf("got %v, want parent — it outranks the child relation the same pair also earns", got)
	}
}

// TestANeighborhoodStaysInsideItsVault. A link by identifier resolves in
// whichever connected vault holds the note, so a neighborhood that does not
// check where one landed draws a note from somewhere else — or, when the two
// vaults file a note at the same path, the wrong note under the right name.
func TestANeighborhoodStaysInsideItsVault(t *testing.T) {
	t.Parallel()
	const shared = "notes/Entropy.md"
	db, here := newIndexedVault(t, map[string]string{
		shared:    "---\ntitle: Entropy here\nid: 01M02ACGM0FYMSXNDP29C90JN1\n---\n\n# Entropy here\n",
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"note://01M02ACGM0FYMSXNDP29C90JN2\"\n    role: child\n---\n\n# Area\n",
	})
	addVault(t, db, testsupport.NewVault(t, map[string]string{
		shared: "---\ntitle: Entropy elsewhere\nid: 01M02ACGM0FYMSXNDP29C90JN2\n---\n\n# Entropy elsewhere\n",
	}))

	n, err := query.NewShowNeighborhood(db.NoteQueries(), db.NoteQueries()).
		Execute(t.Context(), here, "Area.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := relationsOf(n); len(got) != 0 {
		t.Errorf("got %v — the note that identifier names is in another vault", got)
	}
}

// TestOnlyNavigableLinksTakeARelation. A wikilink in prose and an attachment are
// links, and neither is a place in the hierarchy.
func TestOnlyNavigableLinksTakeARelation(t *testing.T) {
	t.Parallel()
	n := neighborhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"https://example.org/paper\"\n    role: attachment\n" +
			"  - to: \"[[Idea]]\"\n    role: jump\n---\n\n# Area\n\nMentioned in prose: [[Other]].\n",
		"Idea.md":  "---\ntitle: Idea\n---\n\n# Idea\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n",
	}, "Area.md")

	if got := relationsOf(n); !slices.Equal(got, []string{"jump Idea.md"}) {
		t.Errorf("got %v — prose links and attachments are not relations", got)
	}
}

// TestADanglingLinkHasNoRelation: there is nothing to draw and nowhere to go.
func TestADanglingLinkHasNoRelation(t *testing.T) {
	t.Parallel()
	n := neighborhoodOf(t, map[string]string{
		"Area.md": "---\ntitle: Area\nlinks:\n  - to: \"[[Nothing by that name]]\"\n    role: child\n---\n\n# Area\n",
	}, "Area.md")

	if got := relationsOf(n); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
	if n.Focus.Title != "Area" {
		t.Errorf("focus = %+v", n.Focus)
	}
}

// TestTheSameVaultDrawsTheSameWayTwice.
//
// Every note has the same name, on purpose. An ordering by note name is
// undefined when notes share a name, and a vault where that is true shows the
// problem: twelve daily notes, twelve chapters, twelve "Notes".
func TestTheSameVaultDrawsTheSameWayTwice(t *testing.T) {
	t.Parallel()
	files := map[string]string{"Area.md": "---\ntitle: Area\n---\n\n# Area\n"}
	for i := range 12 {
		files[fmt.Sprintf("Child%02d.md", i)] = fmt.Sprintf(
			"---\ntitle: Notes\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n---\n\n# Notes %02d\n", i)
	}

	first := relationsOf(neighborhoodOf(t, files, "Area.md"))
	for range 8 {
		if again := relationsOf(neighborhoodOf(t, files, "Area.md")); !slices.Equal(first, again) {
			t.Fatalf("two runs disagree:\n  %v\n  %v", first, again)
		}
	}
	if len(first) != 12 {
		t.Errorf("%d children, want 12", len(first))
	}
}

// TestAVaultOpensOnItsFirstNote.
func TestAVaultOpensOnItsFirstNote(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, map[string]string{
		"Area.md": "---\ntitle: Area\n---\n\n# Area\n",
		"Idea.md": "---\ntitle: Idea\n---\n\n# Idea\n",
	})

	opening, found, err := db.NoteQueries().GetOpeningNote(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("a vault with two notes opens on neither")
	}
	if opening.Title == "" || opening.Path == "" {
		t.Errorf("opening = %+v", opening)
	}

	again, _, err := db.NoteQueries().GetOpeningNote(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again != opening {
		t.Errorf("a vault opens somewhere different each time: %+v then %+v", opening, again)
	}
}

func TestAnEmptyVaultOpensOnNothing(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, map[string]string{})

	_, found, err := db.NoteQueries().GetOpeningNote(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("an empty vault opened on something")
	}
}
