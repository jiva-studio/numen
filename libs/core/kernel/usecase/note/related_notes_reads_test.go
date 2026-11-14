package note_test

import (
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// countingLinks counts the calls to the link batches and the paths requested in them.
type countingLinks struct {
	graph
	linkCalls     int
	backlinkCalls int
	links         map[string]int
	backlinks     map[string]int
}

func newCountingLinks(g graph) *countingLinks {
	return &countingLinks{graph: g, links: map[string]int{}, backlinks: map[string]int{}}
}

func (c *countingLinks) GetLinksOf(ctx context.Context, v domain.VaultID, paths []string) (map[string][]domain.ResolvedLink, error) {
	c.linkCalls++
	for _, p := range paths {
		c.links[p]++
	}
	return c.graph.GetLinksOf(ctx, v, paths)
}

func (c *countingLinks) GetBacklinksOf(ctx context.Context, v domain.VaultID, paths []string) (map[string][]domain.ResolvedLink, error) {
	c.backlinkCalls++
	for _, p := range paths {
		c.backlinks[p]++
	}
	return c.graph.GetBacklinksOf(ctx, v, paths)
}

func TestEachListOfLinksIsReadOnceAtDepthTwo(t *testing.T) {
	t.Parallel()
	counted := newCountingLinks(depthGraph())

	_, err := query.NewListRelatedNotes(counted, &pathLookup{}).Execute(
		t.Context(), domain.Vault{ID: graphVault}, "F.md", domain.DepthFar)
	if err != nil {
		t.Fatal(err)
	}

	for path, n := range counted.links {
		if n != 1 {
			t.Errorf("the links of %s were read %d times, want once", path, n)
		}
	}
	for path, n := range counted.backlinks {
		if n != 1 {
			t.Errorf("the backlinks of %s were read %d times, want once", path, n)
		}
	}
}

func fanGraph(owners int) graph {
	g := graph{out: map[string][]domain.ResolvedLink{}, in: map[string][]domain.ResolvedLink{}}
	for i := range owners {
		owner, item := "O"+strconv.Itoa(i)+".md", "I"+strconv.Itoa(i)+".md"
		g.out["F.md"] = append(g.out["F.md"], createOutgoingLink(owner, domain.RoleChild, ""))
		g.out[owner] = []domain.ResolvedLink{createOutgoingLink(item, domain.RoleChild, "")}
	}
	return g
}

func TestTheCallsToTheLinksDoNotGrowWithTheNotesRelatedToTheFocus(t *testing.T) {
	t.Parallel()
	calls := func(owners int) (links, backlinks int) {
		counted := newCountingLinks(fanGraph(owners))
		_, err := query.NewListRelatedNotes(counted, &pathLookup{}).Execute(
			t.Context(), domain.Vault{ID: graphVault}, "F.md", domain.DepthFar)
		if err != nil {
			t.Fatal(err)
		}
		return counted.linkCalls, counted.backlinkCalls
	}

	fewLinks, fewBacklinks := calls(2)
	manyLinks, manyBacklinks := calls(50)

	if fewLinks != manyLinks || fewBacklinks != manyBacklinks {
		t.Errorf("2 related notes: %d and %d calls, 50 related notes: %d and %d calls", fewLinks, fewBacklinks, manyLinks, manyBacklinks)
	}
}

func TestANoteSharedByManyOwnersIsReadOnce(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{
		"F.md": {createOutgoingLink("A.md", domain.RoleChild, ""), createOutgoingLink("B.md", domain.RoleChild, "")},
		"A.md": {createOutgoingLink("X.md", domain.RoleChild, "")},
		"B.md": {createOutgoingLink("X.md", domain.RoleChild, "")},
	}}
	refs := &pathLookup{}

	showDepth(t, g, refs, "F.md", domain.DepthFar)

	n := 0
	for _, p := range refs.paths {
		if p == "X.md" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("X.md was requested %d times in %v, want once", n, refs.paths)
	}
}

func TestADepthTwoItemMayBeARelatedNoteOfTheFocusButNeverTheFocusOrItsOwner(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{
		"F.md": {createOutgoingLink("A.md", domain.RoleChild, ""), createOutgoingLink("B.md", domain.RoleChild, "")},
		"A.md": {
			createOutgoingLink("B.md", domain.RoleJump, ""),
			createOutgoingLink("A.md", domain.RoleJump, ""),
			createOutgoingLink("X.md", domain.RoleJump, ""),
		},
	}}

	got := showDepth(t, g, &pathLookup{}, "F.md", domain.DepthFar)

	if want := []string{"jump B.md", "jump X.md"}; !slices.Equal(itemsOf(got, "A.md"), want) {
		t.Errorf("items of A %v, want %v", itemsOf(got, "A.md"), want)
	}
}

func TestAReferenceReplacedByAStrongerRelationHasNoDirection(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {createOutgoingLink("A.md", domain.RoleRef, "")},
		},
		in: map[string][]domain.ResolvedLink{
			"F.md": {writtenIn("A.md", domain.RoleParent)},
		},
	}

	got := showDepth(t, g, &pathLookup{}, "F.md", domain.DepthNear)

	if len(got.Related) != 1 || got.Related[0].Relation != domain.RelationChild || got.Related[0].Direction != "" {
		t.Errorf("related = %+v, want A.md as a child with no direction", got.Related)
	}
}
