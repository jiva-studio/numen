package note_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func describeRelated(n domain.NoteGraph) []string {
	out := make([]string, 0, len(n.Related))
	for _, r := range n.Related {
		line := string(r.Relation) + " " + r.Path
		if r.Direction != "" {
			line += " " + string(r.Direction)
		}
		out = append(out, line)
	}
	return out
}

func checkRelatedNotes(t *testing.T, got domain.NoteGraph, want []string) {
	t.Helper()
	if described := describeRelated(got); !slices.Equal(described, want) {
		t.Errorf("got %v, want %v", described, want)
	}
}

func textNote(title, body string) string {
	return "---\ntitle: " + title + "\n---\n\n# " + title + "\n\n" + body + "\n"
}

func TestANoteLinkedInTheTextOfTheFocusIsAnOutgoingReference(t *testing.T) {
	t.Parallel()
	got := relatedNotesOf(t, map[string]string{
		"Focus.md": textNote("Focus", "Compare with [[Alpha]]."),
		"Alpha.md": textNote("Alpha", "Nothing here."),
	}, "Focus.md")
	checkRelatedNotes(t, got, []string{"reference Alpha.md outgoing"})
	if len(got.Connections) != 0 {
		t.Errorf("a reference is drawn by the client, and the kernel made lines %v", linesOf(got))
	}
}

func TestANoteThatLinksToTheFocusInItsTextIsAnIncomingReference(t *testing.T) {
	t.Parallel()
	got := relatedNotesOf(t, map[string]string{
		"Focus.md": textNote("Focus", "Links to nothing."),
		"Beta.md":  textNote("Beta", "See [[Focus]]."),
	}, "Focus.md")
	checkRelatedNotes(t, got, []string{"reference Beta.md incoming"})
}

func TestARoleRefEntryOfTheFrontmatterIsAReference(t *testing.T) {
	t.Parallel()
	got := relatedNotesOf(t, map[string]string{
		"Focus.md": "---\ntitle: Focus\nlinks:\n  - to: \"[[Alpha]]\"\n    role: ref\n---\n\n# Focus\n",
		"Alpha.md": textNote("Alpha", "Nothing here."),
	}, "Focus.md")
	checkRelatedNotes(t, got, []string{"reference Alpha.md outgoing"})
}

func TestAReferenceToAMissingNoteIsNotShown(t *testing.T) {
	t.Parallel()
	got := relatedNotesOf(t, map[string]string{
		"Focus.md": textNote("Focus", "Gone: [[Nowhere]]. Here: [[Alpha]]. Itself: [[Focus]]."),
		"Alpha.md": textNote("Alpha", "Nothing here."),
	}, "Focus.md")
	checkRelatedNotes(t, got, []string{"reference Alpha.md outgoing"})
}

func TestAnExplicitRelationBeatsATextLinkToTheSameNote(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {
				createOutgoingLink("P.md", domain.RoleParent, ""),
				createOutgoingLink("C.md", domain.RoleChild, ""),
				createOutgoingLink("J.md", domain.RoleJump, ""),
				createOutgoingLink("P.md", domain.RoleRef, ""),
				createOutgoingLink("C.md", domain.RoleRef, ""),
				createOutgoingLink("J.md", domain.RoleRef, ""),
			},
		},
		in: map[string][]domain.ResolvedLink{
			"F.md": {
				writtenIn("J.md", domain.RoleRef),
				writtenIn("C.md", domain.RoleRef),
				writtenIn("P.md", domain.RoleRef),
			},
		},
	}

	got, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, got, []string{"parent P.md", "child C.md", "jump J.md"})
}

func TestAnExplicitRelationWrittenInTheOtherNoteBeatsATextLink(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {
				createOutgoingLink("C.md", domain.RoleRef, ""),
			},
		},
		in: map[string][]domain.ResolvedLink{
			"F.md": {writtenIn("C.md", domain.RoleChild)},
		},
	}

	got, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, got, []string{"parent C.md"})
}

func TestANoteLinkedBothWaysIsOneReferenceWithBothDirections(t *testing.T) {
	t.Parallel()
	got := relatedNotesOf(t, map[string]string{
		"Focus.md": textNote("Focus", "See [[Alpha]]."),
		"Alpha.md": textNote("Alpha", "Back to [[Focus]]."),
	}, "Focus.md")
	checkRelatedNotes(t, got, []string{"reference Alpha.md both"})
}

func TestTwoLinksInTheSameDirectionStayThatDirection(t *testing.T) {
	t.Parallel()
	g := graph{out: map[string][]domain.ResolvedLink{"F.md": {
		createOutgoingLink("A.md", domain.RoleRef, ""),
		createOutgoingLink("A.md", domain.RoleRef, ""),
	}}}

	got, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, got, []string{"reference A.md outgoing"})
}

func TestLinksThatLeadNowhereInTheVaultAreNotReferences(t *testing.T) {
	t.Parallel()
	elsewhere := createOutgoingLink("X.md", domain.RoleRef, "")
	elsewhere.ToVault = "other"
	dangling := createOutgoingLink("Y.md", domain.RoleRef, "")
	dangling.ToVault = ""
	g := graph{out: map[string][]domain.ResolvedLink{"F.md": {
		elsewhere,
		dangling,
		createOutgoingLink("F.md", domain.RoleRef, ""),
		createOutgoingLink("Gone.md", domain.RoleRef, ""),
		createOutgoingLink("A.md", domain.RoleRef, ""),
	}}}

	got, err := showGraph(g, &pathLookup{unknown: "Gone.md"}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, got, []string{"reference A.md outgoing"})
}

func TestASiblingLinkedInTheTextIsShownOnceAsASibling(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {
				createOutgoingLink("P.md", domain.RoleParent, ""),
				createOutgoingLink("S.md", domain.RoleRef, ""),
			},
			"P.md": {createOutgoingLink("S.md", domain.RoleChild, "")},
		},
		in: map[string][]domain.ResolvedLink{
			"F.md": {writtenIn("S.md", domain.RoleRef)},
			"P.md": {writtenIn("F.md", domain.RoleParent)},
		},
	}

	got, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, got, []string{"parent P.md", "sibling S.md"})
}

func TestReferencesFollowTheLinksOfTheFocusThenTheBacklinks(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{"F.md": {
			createOutgoingLink("A.md", domain.RoleRef, ""),
			createOutgoingLink("J.md", domain.RoleJump, ""),
			createOutgoingLink("B.md", domain.RoleRef, ""),
		}},
		in: map[string][]domain.ResolvedLink{"F.md": {
			writtenIn("D.md", domain.RoleRef),
			writtenIn("C.md", domain.RoleRef),
		}},
	}
	want := []string{
		"reference A.md outgoing", "jump J.md", "reference B.md outgoing",
		"reference D.md incoming", "reference C.md incoming",
	}

	first, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	second, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, first, want)
	checkRelatedNotes(t, second, want)
}

func TestTheNotesAParentReferencesAreNotSiblingsOrReferencesOfTheFocus(t *testing.T) {
	t.Parallel()
	g := graph{
		out: map[string][]domain.ResolvedLink{
			"F.md": {createOutgoingLink("P.md", domain.RoleParent, "")},
			"P.md": {createOutgoingLink("R.md", domain.RoleRef, "")},
		},
		in: map[string][]domain.ResolvedLink{"P.md": {writtenIn("F.md", domain.RoleParent)}},
	}

	got, err := showGraph(g, &pathLookup{}, "F.md")
	if err != nil {
		t.Fatal(err)
	}
	checkRelatedNotes(t, got, []string{"parent P.md"})
}
