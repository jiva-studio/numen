package markdown

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Everything under `links:` but a block sequence is refused: an entry is
// replaced on its own, and nothing else there is laid out a line at a time.
func TestLinksThatAreNotABlockSequenceAreRefused(t *testing.T) {
	for name, raw := range map[string]string{
		"a flow mapping":  "---\nid: 01\nlinks: {}\ntype: deck\n---\nbody\n",
		"a filled one":    "---\nid: 01\nlinks: {a: b}\ntype: deck\n---\nbody\n",
		"a flow sequence": "---\nid: 01\nlinks: [{to: A, role: jump}]\n---\nbody\n",
		"a scalar":        "---\nid: 01\nlinks: none\n---\nbody\n",
		"a written null":  "---\nid: 01\nlinks: ~\ntype: deck\n---\nbody\n",
		"the word null":   "---\nid: 01\nlinks: null\ntype: deck\n---\nbody\n",
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
				t.Errorf("a refused write changed the note\n want %q\n  got %q", raw, got)
			}
		})
	}
}

// A `links:` key holding nothing is where a first entry is written.
func TestAnEmptyLinksKeyTakesTheFirstEntry(t *testing.T) {
	d, err := Open([]byte("---\nid: 01\nlinks:\ntype: deck\n---\nbody\n"))
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
	} else if links := Parse(domain.FileRef{Path: "D.md"}, []byte(again.Bytes())).Links; len(links) != 1 {
		t.Errorf("links = %v\n%s", links, got)
	}
}
