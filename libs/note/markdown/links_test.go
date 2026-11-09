package markdown_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
)

func parse(t *testing.T, body string) domain.Note {
	t.Helper()
	return markdown.Parse(domain.FileStat{Path: "notes/Source.md"}, []byte(body))
}

func targets(n domain.Note) []string {
	var out []string
	for _, l := range n.Links {
		out = append(out, string(l.Role)+" "+l.Target.String())
	}
	return out
}

func TestWikilinksBecomeReferences(t *testing.T) {
	n := parse(t, "See [[Entropy]] and [[Thermodynamics]].\n")
	want := []string{"ref name://Entropy", "ref name://Thermodynamics"}
	if !slices.Equal(targets(n), want) {
		t.Errorf("got %v, want %v", targets(n), want)
	}
}

func TestAnAliasAndABlockAreNotPartOfTheTarget(t *testing.T) {
	// An alias is how the link is read in the sentence and a fragment names a
	// place inside the note. Neither is part of the target.
	n := parse(t, "[[Entropy|the entropy note]] and [[Entropy#^a7f3d21e]]\n")
	if got := targets(n); !slices.Equal(got, []string{"ref name://Entropy"}) {
		t.Errorf("got %v", got)
	}
}

func TestALinkInsideACodeFenceIsAnExample(t *testing.T) {
	n := parse(t, "```\n[[Not A Link]]\n```\n\n[[Real]]\n")
	if got := targets(n); !slices.Equal(got, []string{"ref name://Real"}) {
		t.Errorf("got %v", got)
	}
}

func TestSchemesAreReadAndTitlesAreNot(t *testing.T) {
	for _, c := range []struct{ rawAddress, want string }{
		{"[[Entropy]]", "name://Entropy"},
		{"note://01M02ACGM0FYMSXNDP29C90JNR", "note://01M02ACGM0FYMSXNDP29C90JNR"},
		{"asset://a1b2c3d4", "asset://a1b2c3d4"},
		{"https://example.org/paper", "https://example.org/paper"},
		// A colon in a title is not a scheme, which is the whole reason an
		// address is stored with one.
		{"[[Lecture 3: entropy]]", "name://Lecture 3: entropy"},
	} {
		if got := domain.ParseAddress(c.rawAddress).String(); got != c.want {
			t.Errorf("%q parsed to %q, want %q", c.rawAddress, got, c.want)
		}
	}
}

func TestAnnotatedLinksCarryRoleTypeAndArgument(t *testing.T) {
	n := parse(t, `---
links:
  - to: "[[Thermodynamics]]"
    role: parent
    type: requires
    note: only eigenvectors are needed
---

body
`)
	if len(n.Links) != 1 {
		t.Fatalf("got %v", targets(n))
	}
	l := n.Links[0]
	if l.Role != domain.RoleParent || l.Type != "requires" || l.Why == "" {
		t.Errorf("got %+v", l)
	}
}

func TestAnAnnotatedLinkWinsOverTheSameLinkInProse(t *testing.T) {
	// The same target written in both places is one link, and the record with more
	// information is the one that survives.
	n := parse(t, `---
links:
  - to: "[[Entropy]]"
    role: parent
---

Mentioned again as [[Entropy]] in the text.
`)
	if len(n.Links) != 1 {
		t.Fatalf("got %v", targets(n))
	}
	if n.Links[0].Role != domain.RoleParent {
		t.Errorf("the prose mention overwrote the described link: %+v", n.Links[0])
	}
}

func TestALinkWithoutARoleIsReportedInsteadOfGuessed(t *testing.T) {
	n := parse(t, `---
links:
  - to: "[[Entropy]]"
---

body
`)
	if len(n.Links) != 0 {
		t.Errorf("a link with no role was stored: %v", targets(n))
	}
	if len(n.Problems) != 1 {
		t.Errorf("problems = %v, want one", n.Problems)
	}
}

func TestAnUnknownRoleIsRefused(t *testing.T) {
	// The list of roles is closed: navigation reads it, so a role with no decided
	// behavior has none, and storing it would misrepresent that.
	n := parse(t, `---
links:
  - to: "[[Entropy]]"
    role: sideways
---

body
`)
	if len(n.Links) != 0 {
		t.Errorf("an unknown role was stored: %v", targets(n))
	}
	if len(n.Problems) != 1 || !strings.Contains(n.Problems[0], "sideways") {
		t.Errorf("problems = %v", n.Problems)
	}
}

func TestTheIdentifierIsReadFromFrontmatter(t *testing.T) {
	n := parse(t, "---\nid: 01M02ACGM0FYMSXNDP29C90JNR\n---\n\nbody\n")
	if n.ID != "01M02ACGM0FYMSXNDP29C90JNR" {
		t.Errorf("id = %q", n.ID)
	}
}

func TestANoteWithoutAnIdentifierIsStillANote(t *testing.T) {
	n := parse(t, "# Title\n\nbody\n")
	if n.ID != "" {
		t.Errorf("id = %q, want empty", n.ID)
	}
	if n.Title != "Source" || n.Body == "" {
		t.Error("the rest of the note was not parsed")
	}
}

func TestLinksEdgeCasesAndProblems(t *testing.T) {
	// Leading empty line followed by link and fenced link, plus duplicate link before second link on same line.
	n := parse(t, "\n[[First]] [[First]] [[Second]]\n```\n[[Ignored]]\n```\n")
	if got := targets(n); !slices.Equal(got, []string{"ref name://First", "ref name://Second"}) {
		t.Errorf("got %v", got)
	}

	// Frontmatter link problems with a valid entry after each invalid entry.
	nProblems := parse(t, `---
id: 123
title: "   "
links:
  - 42
  - role: parent
  - to: "[[MissingRole]]"
  - to: "[[BadMarker]]"
    role: jump
    marker: unknown
  - to: "[[Valid]]"
    role: child
---
body
`)
	if got := targets(nProblems); !slices.Equal(got, []string{"child name://Valid"}) {
		t.Errorf("got links %v", got)
	}
	wantProblems := []string{
		"links: entry is not a mapping",
		"links: an entry has no target",
		"links: [[MissingRole]] has no role",
		"links: [[BadMarker]] has an unknown marker unknown",
		"id is not text",
	}
	if !slices.Equal(nProblems.Problems, wantProblems) {
		t.Errorf("problems = %v, want %v", nProblems.Problems, wantProblems)
	}
	if nProblems.Title != "Source" {
		t.Errorf("whitespace title did not fall back to filename: %q", nProblems.Title)
	}

	nBadID := parse(t, "---\nid: not-a-ulid\n---\nbody\n")
	if !slices.Equal(nBadID.Problems, []string{"id not-a-ulid is not a ULID"}) {
		t.Errorf("bad id problems = %v", nBadID.Problems)
	}

	nScalarFM := parse(t, "---\na\n---\nbody\n")
	if nScalarFM.FrontmatterErr == "" {
		t.Error("expected FrontmatterErr for 1-char non-mapping frontmatter")
	}
}
