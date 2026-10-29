package note_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
)

const graphVault domain.VaultID = "graph"

// graph is a hand-written link set for a vault: the outgoing and incoming links
// of each note, in return order.
type graph struct {
	noteport.LinkQueries
	out     map[string][]domain.ResolvedLink
	in      map[string][]domain.ResolvedLink
	failOut string
	failIn  string
	err     error
}

func (g graph) GetLinks(_ context.Context, _ domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	if from == g.failOut {
		return nil, g.err
	}
	return g.out[from], nil
}

func (g graph) GetBacklinks(_ context.Context, _ domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	if to == g.failIn {
		return nil, g.err
	}
	return g.in[to], nil
}

// writes is a link of the note at from, to the note at to.
func createOutgoingLink(to string, role domain.LinkRole, label string) domain.ResolvedLink {
	return domain.ResolvedLink{
		Link: domain.Link{Role: role, Label: label},
		To:   to, ToVault: graphVault,
	}
}

// writtenIn returns a link written by the note at from, as seen from the note
// it points at.
func writtenIn(from string, role domain.LinkRole) domain.ResolvedLink {
	return domain.ResolvedLink{Link: domain.Link{Role: role}, From: from}
}

// asked is a note lookup that returns every path except those marked unknown,
// and records the paths requested.
type asked struct {
	err     error
	unknown string
	paths   []string
}

func (a *asked) GetNotes(_ context.Context, _ domain.VaultID, paths []string) (map[string]domain.NoteRef, error) {
	if a.err != nil {
		return nil, a.err
	}
	a.paths = slices.Clone(paths)
	out := map[string]domain.NoteRef{}
	for _, p := range paths {
		if p != a.unknown {
			out[p] = domain.NoteRef{Path: p}
		}
	}
	return out, nil
}

func showGraph(g graph, refs *asked, focus string) (domain.Neighbourhood, error) {
	return note.NewShowNeighbourhood(g, refs).Execute(
		context.Background(), domain.Vault{ID: graphVault}, focus)
}

// Siblings are the other children of a parent, and only of a parent. A child's
// children and a parent's own parents and jumps are not siblings, and each note
// shown is requested once.
func TestOnlyTheOtherChildrenOfAParentAreSiblings(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {
				createOutgoingLink("C.md", domain.RoleChild, ""),
				createOutgoingLink("P.md", domain.RoleParent, ""),
			},
			"C.md": {createOutgoingLink("G.md", domain.RoleChild, "")},
			"P.md": {
				createOutgoingLink("GP.md", domain.RoleParent, ""),
				createOutgoingLink("J.md", domain.RoleJump, ""),
				createOutgoingLink("S.md", domain.RoleChild, ""),
			},
		},
		in: map[string][]domain.ResolvedLink{
			"P.md": {writtenIn("F.md", domain.RoleParent)},
		},
	}
	refs := &asked{}

	got, err := showGraph(g, refs, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"child C.md", "parent P.md", "sibling S.md"}
	if relations := relationsOf(got); !slices.Equal(relations, want) {
		t.Errorf("got %v, want %v", relations, want)
	}
	if want := []string{"F.md", "C.md", "P.md", "S.md"}; !slices.Equal(refs.paths, want) {
		t.Errorf("the notes asked for are %v, want %v", refs.paths, want)
	}
}

// Whatever fails in finding the neighbours is the answer.
func TestANeighbourhoodThatCannotBeFoundReturnsWhy(t *testing.T) {
	t.Parallel()
	boom := errors.New("the index would not answer")
	parentOfF := map[string][]domain.ResolvedLink{"F.md": {createOutgoingLink("P.md", domain.RoleParent, "")}}
	for name, c := range map[string]struct {
		g    graph
		refs *asked
	}{
		"the links of the note": {
			g:    graph{failOut: "F.md", err: boom},
			refs: &asked{},
		},
		"the links pointing at the note": {
			g:    graph{failIn: "F.md", err: boom},
			refs: &asked{},
		},
		"the links of a parent": {
			g:    graph{out: parentOfF, failOut: "P.md", err: boom},
			refs: &asked{},
		},
		"the notes themselves": {
			g:    graph{out: parentOfF},
			refs: &asked{err: boom},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := showGraph(c.g, c.refs, "F.md"); !errors.Is(err, boom) {
				t.Errorf("got %v", err)
			}
		})
	}
}

// A note unknown to the vault is the center of no neighbourhood, regardless of
// which notes link to it.
func TestANoteTheVaultDoesNotKnowHasNoNeighbourhood(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{"F.md": {createOutgoingLink("C.md", domain.RoleChild, "")}}}

	got, err := showGraph(g, &asked{unknown: "F.md"}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	if got.Focus.Path != "" || len(got.Related) != 0 {
		t.Errorf("got %+v", got)
	}
}

// A link with no place in the hierarchy relates no notes from either end, and a
// note named from both ends does not hide the notes after it.
func TestShowGraph_RelatesOnlyHierarchyRolesFromIncoming(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md":  {createOutgoingLink("P1.md", domain.RoleParent, "")},
			"P1.md": {createOutgoingLink("F.md", domain.RoleChild, "")},
		},
		in: map[string][]domain.ResolvedLink{
			"F.md": {
				writtenIn("W.md", domain.RoleRef),
				writtenIn("P1.md", domain.RoleChild),
				writtenIn("X.md", domain.RoleParent),
			},
			"P1.md": {writtenIn("F.md", domain.RoleParent)},
		},
	}

	got, err := showGraph(g, &asked{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"parent P1.md", "child X.md"}
	if relations := relationsOf(got); !slices.Equal(relations, want) {
		t.Errorf("got %v, want %v", relations, want)
	}
	if !got.Related[0].IsMutual {
		t.Error("both ends name the edge and it is not drawn as mutual")
	}
}

// A relationship written twice at the same rank is drawn as the first one
// wrote it.
func TestTheFirstOfTwoEqualLinksIsTheOneGenerated(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{"F.md": {
		createOutgoingLink("C.md", domain.RoleChild, "first"),
		createOutgoingLink("C.md", domain.RoleChild, "second"),
	}}}

	got, err := showGraph(g, &asked{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Related) != 1 || got.Related[0].Label != "first" {
		t.Errorf("got %+v", got.Related)
	}
}
