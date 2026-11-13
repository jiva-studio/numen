package note_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func showDepth(t *testing.T, g graph, refs *pathLookup, focus string, depth domain.RelationDepth) domain.NoteGraph {
	t.Helper()
	got, err := queryRelated(g, refs, focus, depth)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func itemsOf(n domain.NoteGraph, relatedTo string) []string {
	var out []string
	for _, r := range n.Related {
		if r.RelatedTo == relatedTo {
			out = append(out, string(r.Relation)+" "+r.Path)
		}
	}
	return out
}

func depthGraph() graph {
	return graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {
				createOutgoingLink("C.md", domain.RoleChild, ""),
				createOutgoingLink("P.md", domain.RoleParent, ""),
			},
			"C.md": {
				createOutgoingLink("G.md", domain.RoleChild, ""),
				createOutgoingLink("F.md", domain.RoleParent, ""),
			},
			"P.md": {
				createOutgoingLink("GP.md", domain.RoleParent, ""),
				createOutgoingLink("J.md", domain.RoleJump, ""),
				createOutgoingLink("S.md", domain.RoleChild, ""),
			},
			"S.md": {createOutgoingLink("SC.md", domain.RoleChild, "")},
		},
		in: map[string][]domain.ResolvedLink{
			"P.md": {writtenIn("F.md", domain.RoleParent)},
			"S.md": {writtenIn("P.md", domain.RoleChild)},
		},
	}
}

func TestDepthTwoAddsTheNotesRelatedToEachRelatedNote(t *testing.T) {
	t.Parallel()
	got := showDepth(t, depthGraph(), &pathLookup{}, "F.md", 2)

	if want := []string{"child C.md", "parent P.md", "sibling S.md"}; !slices.Equal(itemsOf(got, ""), want) {
		t.Errorf("notes related to the focus %v, want %v", itemsOf(got, ""), want)
	}
	if want := []string{"child G.md"}; !slices.Equal(itemsOf(got, "C.md"), want) {
		t.Errorf("notes related to the child %v, want %v", itemsOf(got, "C.md"), want)
	}
	if want := []string{"parent GP.md", "jump J.md", "child S.md"}; !slices.Equal(itemsOf(got, "P.md"), want) {
		t.Errorf("notes related to the parent %v, want %v", itemsOf(got, "P.md"), want)
	}
}

func TestDepthTwoItemsOfASiblingAreNeverSiblings(t *testing.T) {
	t.Parallel()
	got := showDepth(t, depthGraph(), &pathLookup{}, "F.md", 2)

	if want := []string{"child SC.md", "parent P.md"}; !slices.Equal(itemsOf(got, "S.md"), want) {
		t.Errorf("notes related to the sibling %v, want %v", itemsOf(got, "S.md"), want)
	}
	for _, r := range got.Related {
		if r.RelatedTo != "" && r.Relation == domain.RelationSibling {
			t.Errorf("%s is a sibling of %s: siblings are only of the focus", r.Path, r.RelatedTo)
		}
	}
}

func TestDepthTwoNeverReturnsTheFocus(t *testing.T) {
	t.Parallel()
	got := showDepth(t, depthGraph(), &pathLookup{}, "F.md", 2)

	for _, r := range got.Related {
		if r.Path == "F.md" {
			t.Errorf("the focus is an item related to %q", r.RelatedTo)
		}
	}
}

func TestANoteIsAnItemOfEveryNoteItIsRelatedTo(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{
		"F.md": {createOutgoingLink("A.md", domain.RoleChild, ""), createOutgoingLink("B.md", domain.RoleChild, "")},
		"A.md": {createOutgoingLink("X.md", domain.RoleChild, "")},
		"B.md": {createOutgoingLink("X.md", domain.RoleChild, "")},
	}}
	got := showDepth(t, g, &pathLookup{}, "F.md", 2)

	if a, b := itemsOf(got, "A.md"), itemsOf(got, "B.md"); !slices.Equal(a, []string{"child X.md"}) || !slices.Equal(b, []string{"child X.md"}) {
		t.Errorf("items of A %v and of B %v, want X in both", a, b)
	}
}

func TestDepthTwoAppliesNoLimit(t *testing.T) {
	t.Parallel()
	var many []domain.ResolvedLink
	for i := range 60 {
		many = append(many, createOutgoingLink("K"+strconv.Itoa(i)+".md", domain.RoleChild, ""))
	}
	g := graph{out: map[string][]domain.ResolvedLink{
		"F.md": {createOutgoingLink("C.md", domain.RoleChild, "")},
		"C.md": many,
	}}
	got := showDepth(t, g, &pathLookup{}, "F.md", 2)

	if n := len(itemsOf(got, "C.md")); n != 60 {
		t.Errorf("%d items of the child, want 60", n)
	}
}

func TestDepthTwoConnectionsEqualThoseOfDepthOne(t *testing.T) {
	t.Parallel()
	one := showDepth(t, depthGraph(), &pathLookup{}, "F.md", 1)
	two := showDepth(t, depthGraph(), &pathLookup{}, "F.md", 2)

	if len(one.Connections) == 0 || len(one.Connections) != len(two.Connections) {
		t.Fatalf("connections %d at depth 1, %d at depth 2", len(one.Connections), len(two.Connections))
	}
	for i := range one.Connections {
		if one.Connections[i].A != two.Connections[i].A || one.Connections[i].B != two.Connections[i].B {
			t.Errorf("connection %d is %v at depth 2, %v at depth 1", i, two.Connections[i], one.Connections[i])
		}
	}
}

func TestDepthOneHasNoItemsOfDepthTwo(t *testing.T) {
	t.Parallel()
	for _, depth := range []domain.RelationDepth{domain.DepthNear} {
		got := showDepth(t, depthGraph(), &pathLookup{}, "F.md", depth)
		for _, r := range got.Related {
			if r.RelatedTo != "" {
				t.Errorf("depth %d returned %s related to %s", depth, r.Path, r.RelatedTo)
			}
		}
	}
}

func TestItemsOfDepthTwoFollowTheNotesRelatedToTheFocusInGroups(t *testing.T) {
	t.Parallel()
	got := showDepth(t, depthGraph(), &pathLookup{}, "F.md", 2)

	var seen []string
	started := false
	for _, r := range got.Related {
		if r.RelatedTo == "" {
			if started {
				t.Fatalf("%s related to the focus comes after an item of depth 2", r.Path)
			}
			continue
		}
		started = true
		if len(seen) == 0 || seen[len(seen)-1] != r.RelatedTo {
			if slices.Contains(seen, r.RelatedTo) {
				t.Errorf("the items of %s are not in one group", r.RelatedTo)
			}
			seen = append(seen, r.RelatedTo)
		}
	}
	if want := []string{"C.md", "P.md", "S.md"}; !slices.Equal(seen, want) {
		t.Errorf("groups in the order %v, want %v", seen, want)
	}
}

func TestNotesMissingFromTheIndexAreDroppedWithTheirItems(t *testing.T) {
	t.Parallel()
	got := showDepth(t, depthGraph(), &pathLookup{unknown: "C.md"}, "F.md", 2)
	for _, r := range got.Related {
		if r.Path == "C.md" || r.RelatedTo == "C.md" {
			t.Errorf("%+v stays though the index lacks C.md", r)
		}
	}

	got = showDepth(t, depthGraph(), &pathLookup{unknown: "GP.md"}, "F.md", 2)
	if want := []string{"jump J.md", "child S.md"}; !slices.Equal(itemsOf(got, "P.md"), want) {
		t.Errorf("items of the parent %v, want %v", itemsOf(got, "P.md"), want)
	}
}

func TestEveryLinkIsReadOnceAndTheNotesAreQueriedOnce(t *testing.T) {
	t.Parallel()
	refs := &pathLookup{}
	showDepth(t, depthGraph(), refs, "F.md", 2)

	if refs.calls != 1 {
		t.Errorf("the notes were queried %d times, want once", refs.calls)
	}
	for _, p := range []string{"G.md", "GP.md", "J.md", "SC.md"} {
		if !slices.Contains(refs.paths, p) {
			t.Errorf("%s is shown by name and was not asked for: %v", p, refs.paths)
		}
	}
}

func TestTheOtherChildrenOfAParentAreItemsOfThatParent(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{
		"F.md": {createOutgoingLink("P.md", domain.RoleParent, "")},
		"P.md": {
			createOutgoingLink("F.md", domain.RoleChild, ""),
			createOutgoingLink("S1.md", domain.RoleChild, ""),
			createOutgoingLink("S2.md", domain.RoleChild, ""),
		},
	}}
	got := showDepth(t, g, &pathLookup{}, "F.md", 2)

	if want := []string{"child S1.md", "child S2.md"}; !slices.Equal(itemsOf(got, "P.md"), want) {
		t.Errorf("items of the parent %v, want %v", itemsOf(got, "P.md"), want)
	}
}

func TestTheParentOfTheFocusIsAnItemOfAReferenceWithTheSameParent(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {createOutgoingLink("P.md", domain.RoleParent, ""), createOutgoingLink("R.md", domain.RoleRef, "")},
			"R.md": {createOutgoingLink("P.md", domain.RoleParent, "")},
		},
		in: map[string][]domain.ResolvedLink{"P.md": {writtenIn("R.md", domain.RoleParent)}},
	}
	got := showDepth(t, g, &pathLookup{}, "F.md", 2)

	if want := []string{"parent P.md"}; !slices.Equal(itemsOf(got, "R.md"), want) {
		t.Errorf("items of the reference %v, want %v", itemsOf(got, "R.md"), want)
	}
	if want := []string{"child R.md"}; !slices.Equal(itemsOf(got, "P.md"), want) {
		t.Errorf("items of the parent %v, want %v", itemsOf(got, "P.md"), want)
	}
}

func TestANoteRelatedToAnOwnerByTwoLinksIsOneItemOfThatOwner(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {createOutgoingLink("A.md", domain.RoleChild, "")},
			"A.md": {createOutgoingLink("X.md", domain.RoleJump, ""), createOutgoingLink("X.md", domain.RoleChild, "")},
		},
	}
	got := showDepth(t, g, &pathLookup{}, "F.md", 2)

	if want := []string{"child X.md"}; !slices.Equal(itemsOf(got, "A.md"), want) {
		t.Errorf("items of A %v, want %v", itemsOf(got, "A.md"), want)
	}
}
