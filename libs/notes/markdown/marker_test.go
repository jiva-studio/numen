package markdown_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

func TestAMarkerIsReadFromARecord(t *testing.T) {
	n := parse(t, "---\nlinks:\n  - to: \"[[Entropy]]\"\n    role: jump\n    marker: arrow\n---\n\nbody\n")
	if len(n.Links) != 1 || n.Links[0].Marker != domain.MarkerArrow {
		t.Fatalf("links = %+v", n.Links)
	}
	if len(n.Problems) != 0 {
		t.Errorf("problems = %v", n.Problems)
	}
}

func TestARecordWithoutAMarkerDrawsNothing(t *testing.T) {
	n := parse(t, "---\nlinks:\n  - to: \"[[Entropy]]\"\n    role: jump\n---\n\nbody\n")
	if len(n.Links) != 1 || n.Links[0].Marker != domain.MarkerNone {
		t.Fatalf("links = %+v", n.Links)
	}
}

func TestAnUnknownMarkerIsAProblemOfTheNote(t *testing.T) {
	n := parse(t, "---\nlinks:\n  - to: \"[[Entropy]]\"\n    role: jump\n    marker: circle\n---\n\nbody\n")
	if len(n.Links) != 0 {
		t.Errorf("a record with an unknown marker was stored: %+v", n.Links)
	}
	if len(n.Problems) != 1 || !strings.Contains(n.Problems[0], "unknown marker circle") {
		t.Errorf("problems = %v", n.Problems)
	}
}

func TestAMarkerRoundTrips(t *testing.T) {
	doc, err := markdown.Open([]byte("---\ntitle: A\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.AddLink(domain.Link{
		Target: domain.ParseAddress("B"), Role: domain.RoleJump, Marker: domain.MarkerArrow,
	}); err != nil {
		t.Fatal(err)
	}
	content := string(doc.Bytes())
	if !strings.Contains(content, "marker: arrow") {
		t.Fatalf("the marker was not written:\n%s", content)
	}
	n := parse(t, content)
	if len(n.Links) != 1 || n.Links[0].Marker != domain.MarkerArrow {
		t.Errorf("links = %+v", n.Links)
	}
}

func TestARecordWithoutAMarkerWritesNoMarkerKey(t *testing.T) {
	doc, err := markdown.Open([]byte("---\ntitle: A\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.AddLink(domain.Link{Target: domain.ParseAddress("B"), Role: domain.RoleJump}); err != nil {
		t.Fatal(err)
	}
	if got := string(doc.Bytes()); strings.Contains(got, "marker") {
		t.Errorf("a record with no marker wrote one:\n%s", got)
	}
}

func TestAnEntryWithAnUnknownMarkerIsNotRewritten(t *testing.T) {
	raw := "---\nlinks:\n  - to: B\n    role: jump\n    marker: circle\n---\n"
	doc, err := markdown.Open([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	err = doc.SetLink(domain.Link{Target: domain.ParseAddress("B"), Role: domain.RoleJump})
	if !errors.Is(err, markdown.ErrNotOwned) {
		t.Fatalf("err = %v, want ErrNotOwned", err)
	}
	if string(doc.Bytes()) != raw {
		t.Errorf("the entry changed:\n%s", doc.Bytes())
	}
}

func TestSetLinkReplacesTheRecordOfATargetAndType(t *testing.T) {
	doc, err := markdown.Open([]byte(
		"---\nlinks:\n  - to: B\n    role: parent\n    type: requires\n    note: why\n    label: old\n" +
			"  - to: B\n    role: ref\n    type: preset\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetLink(domain.Link{
		Target: domain.ParseAddress("B"), Role: domain.RoleJump, Type: "requires", Marker: domain.MarkerArrow,
	}); err != nil {
		t.Fatal(err)
	}
	n := parse(t, string(doc.Bytes()))
	if len(n.Links) != 2 {
		t.Fatalf("links = %+v", n.Links)
	}
	got := n.Links[0]
	if got.Role != domain.RoleJump || got.Marker != domain.MarkerArrow || got.Label != "" || got.Why != "" {
		t.Errorf("the record was not replaced whole: %+v", got)
	}
	if n.Links[1].Type != "preset" {
		t.Errorf("the record of another type changed: %+v", n.Links[1])
	}
}

func TestSetLinkKeepsAnEmptyLabelWithAMarker(t *testing.T) {
	doc, err := markdown.Open([]byte("---\ntitle: A\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetLink(domain.Link{
		Target: domain.ParseAddress("B"), Role: domain.RoleJump, Marker: domain.MarkerArrow,
	}); err != nil {
		t.Fatal(err)
	}
	n := parse(t, string(doc.Bytes()))
	if len(n.Links) != 1 || n.Links[0].Label != "" || n.Links[0].Marker != domain.MarkerArrow {
		t.Errorf("links = %+v", n.Links)
	}
}

func TestSetLinkFindsARecordOfAnotherAddress(t *testing.T) {
	doc, err := markdown.Open([]byte("---\nlinks:\n  - to: notes/B\n    role: child\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.SetLink(
		domain.Link{Target: domain.ParseAddress("B"), Role: domain.RoleJump},
		domain.ParseAddress("notes/B"),
	); err != nil {
		t.Fatal(err)
	}
	n := parse(t, string(doc.Bytes()))
	if len(n.Links) != 1 || n.Links[0].Role != domain.RoleJump {
		t.Errorf("links = %+v", n.Links)
	}
}

func TestRemoveLinkAddressesATypeAndKeepsTheOthers(t *testing.T) {
	doc, err := markdown.Open([]byte(
		"---\nlinks:\n  - to: B\n    role: parent\n  - to: B\n    role: ref\n    type: preset\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	removed, err := doc.RemoveLink(domain.LinkKey{Target: domain.ParseAddress("B")})
	if err != nil || removed != 1 {
		t.Fatalf("removed %d, err %v", removed, err)
	}
	n := parse(t, string(doc.Bytes()))
	if len(n.Links) != 1 || n.Links[0].Type != "preset" {
		t.Errorf("links = %+v", n.Links)
	}
}

func TestTheFixtureVaultContainsARecordWithAMarker(t *testing.T) {
	n := parseFile(t, "Thermodynamics.md")

	var arrows int
	for _, l := range n.Links {
		if l.Marker == domain.MarkerArrow {
			arrows++
		}
	}
	if arrows != 1 {
		t.Errorf("%d records carry an arrow, want 1: %+v", arrows, n.Links)
	}
}

func TestTheFixtureVaultContainsARecordWithAnUnknownMarker(t *testing.T) {
	n := parseFile(t, "edge/broken-links.md")

	var found bool
	for _, p := range n.Problems {
		found = found || strings.Contains(p, "unknown marker circle")
	}
	if !found {
		t.Errorf("problems = %v", n.Problems)
	}
}
