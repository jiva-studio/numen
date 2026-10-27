package markdown

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Everything under `links:` but a block sequence fails: an entry is
// replaced on its own, and nothing else there is laid out a line at a time.
func TestLinksThatAreNotABlockSequenceAreRefused(t *testing.T) {
	for name, raw := range map[string]string{
		"a flow mapping":  "---\nid: 01\nlinks: {}\ntype: sample\n---\nbody\n",
		"a filled one":    "---\nid: 01\nlinks: {a: b}\ntype: sample\n---\nbody\n",
		"a flow sequence": "---\nid: 01\nlinks: [{to: A, role: jump}]\n---\nbody\n",
		"a scalar":        "---\nid: 01\nlinks: none\n---\nbody\n",
		"a written null":  "---\nid: 01\nlinks: ~\ntype: sample\n---\nbody\n",
		"the word null":   "---\nid: 01\nlinks: null\ntype: sample\n---\nbody\n",
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			err = d.AddLink(domain.Link{Target: domain.ParseAddress("P"), Role: domain.RoleRef, Type: "preset"})
			if !errors.Is(err, ErrInline) {
				t.Errorf("want ErrInline, got %v", err)
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("a failed write changed the note\n want %q\n  got %q", raw, got)
			}
		})
	}
}

// A `links:` key holding nothing is where a first entry is written.
func TestAnEmptyLinksKeyConsumesTheFirstEntry(t *testing.T) {
	d, err := Open([]byte("---\nid: 01\nlinks:\ntype: sample\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.AddLink(domain.Link{
		Target: domain.ParseAddress("P"), Role: domain.RoleRef, Type: "preset",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	got := string(d.Bytes())
	if again, err := Open([]byte(got)); err != nil {
		t.Fatalf("the note stopped being readable: %v\n%s", err, got)
	} else if links := Parse(domain.FileStat{Path: "D.md"}, []byte(again.Bytes())).Links; len(links) != 1 {
		t.Errorf("links = %v\n%s", links, got)
	}
}

func TestAddAndUpdateLinkWithLabelAndRole(t *testing.T) {
	doc, err := Open([]byte("---\ntitle: Bhagavad-gita\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// 1. Add child link with label "one"
	if err := doc.AddLink(domain.Link{
		Target: domain.ParseAddress("Q"),
		Role:   domain.RoleChild,
		Label:  "one",
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}
	parsed := Parse(domain.FileStat{Path: "Bhagavad-gita.md"}, doc.Bytes())
	if len(parsed.Links) != 1 || parsed.Links[0].Label != "one" || parsed.Links[0].Role != domain.RoleChild {
		t.Fatalf("unexpected parsed links after add: %+v", parsed.Links)
	}

	// 2. Update existing link to have label "two" and role "jump"
	if err := doc.AddLink(domain.Link{
		Target: domain.ParseAddress("Q"),
		Role:   domain.RoleJump,
		Label:  "two",
	}); err != nil {
		t.Fatalf("update link: %v", err)
	}
	parsed = Parse(domain.FileStat{Path: "Bhagavad-gita.md"}, doc.Bytes())
	if len(parsed.Links) != 1 || parsed.Links[0].Label != "two" || parsed.Links[0].Role != domain.RoleJump {
		t.Fatalf("unexpected parsed links after update: %+v", parsed.Links)
	}

	// 3. Remove link
	if _, err := doc.RemoveLink(domain.ParseAddress("Q"), domain.RoleJump); err != nil {
		t.Fatalf("remove link: %v", err)
	}
	parsed = Parse(domain.FileStat{Path: "Bhagavad-gita.md"}, doc.Bytes())
	if len(parsed.Links) != 0 {
		t.Fatalf("expected 0 links after remove, got: %+v", parsed.Links)
	}
}

// A key written after the `links:` block is not inside the last entry's span,
// a trailing comment on that entry's last line included.
func TestAKeyAfterLinksSurvivesAnEditWithATrailingComment(t *testing.T) {
	raw := "---\nid: 01\nlinks:\n  - to: A\n    role: jump  # a note to self\ntags: [one, two]\n---\nbody\n"
	doc, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := doc.AddLink(domain.Link{Target: domain.ParseAddress("B"), Role: domain.RoleRef}); err != nil {
		t.Fatalf("add link: %v", err)
	}

	got := string(doc.Bytes())
	if !strings.Contains(got, "tags: [one, two]") {
		t.Errorf("tags was lost\n%s", got)
	}
	if again, err := Open([]byte(got)); err != nil {
		t.Fatalf("the note stopped being readable: %v\n%s", err, got)
	} else if links := Parse(domain.FileStat{Path: "D.md"}, again.Bytes()).Links; len(links) != 2 {
		t.Errorf("links = %v\n%s", links, got)
	}
}

func TestAddLinkWithComplexLabelContainingColonsAndQuotes(t *testing.T) {
	doc, err := Open([]byte("---\ntitle: Srimad-Bhagavatam\n---\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	complexLabel := "Part 1: The Creation: Section A (with \"quotes\" & symbols 🚀)"
	if err := doc.AddLink(domain.Link{
		Target: domain.ParseAddress("Canto1"),
		Role:   domain.RoleChild,
		Label:  complexLabel,
	}); err != nil {
		t.Fatalf("add link: %v", err)
	}

	parsed := Parse(domain.FileStat{Path: "Srimad-Bhagavatam.md"}, doc.Bytes())
	if len(parsed.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(parsed.Links))
	}
	if parsed.Links[0].Label != complexLabel {
		t.Errorf("expected label %q, got %q", complexLabel, parsed.Links[0].Label)
	}
	if parsed.Links[0].Role != domain.RoleChild {
		t.Errorf("expected role child, got %v", parsed.Links[0].Role)
	}
}
