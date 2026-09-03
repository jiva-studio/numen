package markdown

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A carriage return on its own is a line break, and so are NEL, LS and PS. The
// lines a write is addressed by are the lines the frontmatter is read in.
func TestALineIsEveryBreakYAMLReads(t *testing.T) {
	for name, one := range map[string]struct {
		raw, want string
		change    func(*Document) error
	}{
		"a return between keys": {
			raw:  "---\ntags:\n  - moss\rlinks:\n  - to: Ferns\n    role: ref\n---\nbody\n",
			want: "---\ntags:\n  - moss\rlinks:\n  - to: Ferns\n    role: jump\n---\nbody\n",
			change: func(d *Document) error {
				_, err := d.UpdateLink(domain.ParseAddress("Ferns"), domain.Link{Role: domain.RoleJump})
				return err
			},
		},
		"a return inside a block scalar": {
			raw: "---\nlinks:\n  - to: Ferns\n    role: ref\n    note: |\n      the first thought\r      the second one\nkeep: me\n---\nbody\n",
			want: "---\nlinks:\n  - to: Ferns\n    role: jump\n    note: |-\n      the first thought\n      the second one\n" +
				"keep: me\n---\nbody\n",
			change: func(d *Document) error {
				_, err := d.UpdateLink(domain.ParseAddress("Ferns"), domain.Link{Role: domain.RoleJump})
				return err
			},
		},
		"a return before a quoted value": {
			raw:  "---\nkeep: me\rlinks:\n  - to: \"Ferns\"\n    role: ref\n---\nbody\n",
			want: "---\nkeep: me\rlinks:\n  - to: \"Mosses\"\n    role: ref\n---\nbody\n",
			change: func(d *Document) error {
				_, err := d.PointLinksAt(domain.ParseAddress("Ferns"), "Mosses")
				return err
			},
		},
		"a return before the last line of the block": {
			raw:    "---\nlinks:\n  - to: Ferns\n    role: ref\rseen: 3\n---\nbody\n",
			want:   "---\nlinks:\n  - to: Ferns\n    role: ref\rseen: 4\n---\nbody\n",
			change: func(d *Document) error { return d.SetValue("seen", 4) },
		},
		"a return and a newline together": {
			raw:  "---\r\ntags:\r\n  - moss\r\nlinks:\r\n  - to: Ferns\r\n    role: ref\r\n---\r\nbody\r\n",
			want: "---\r\ntags:\r\n  - moss\r\nlinks:\r\n  - to: Ferns\r\n    role: jump\r\n---\r\nbody\r\n",
			change: func(d *Document) error {
				_, err := d.UpdateLink(domain.ParseAddress("Ferns"), domain.Link{Role: domain.RoleJump})
				return err
			},
		},
		"a paragraph separator between keys": {
			raw:  "---\ntags:\n  - moss links:\n  - to: Ferns\n    role: ref\n---\nbody\n",
			want: "---\ntags:\n  - moss links:\n  - to: Ferns\n    role: jump\n---\nbody\n",
			change: func(d *Document) error {
				_, err := d.UpdateLink(domain.ParseAddress("Ferns"), domain.Link{Role: domain.RoleJump})
				return err
			},
		},
		"a next line between keys": {
			raw:  "---\ntags:\n  - mosslinks:\n  - to: Ferns\n    role: ref\n---\nbody\n",
			want: "---\ntags:\n  - mosslinks:\n  - to: Ferns\n    role: jump\n---\nbody\n",
			change: func(d *Document) error {
				_, err := d.UpdateLink(domain.ParseAddress("Ferns"), domain.Link{Role: domain.RoleJump})
				return err
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(one.raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := one.change(d); err != nil {
				t.Fatalf("change: %v", err)
			}
			if got := string(d.Bytes()); got != one.want {
				t.Errorf("want %q\n got %q", one.want, got)
			}
		})
	}
}

// The note that was overwritten: a return stands between the `links:` key and
// its one entry, and the top-level key below the block is not the block's.
func TestAReturnDoesNotMoveTheLinksBlockDown(t *testing.T) {
	raw := "---\nlinks:\r- to: Ferns\nseen: 3\n---\nbody\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := d.UpdateLink(domain.ParseAddress("Ferns"), domain.Link{Role: domain.RoleRef}); err != nil {
		t.Fatalf("update: %v", err)
	}
	want := "---\nlinks:\r- to: Ferns\n  role: ref\nseen: 3\n---\nbody\n"
	if got := string(d.Bytes()); got != want {
		t.Errorf("want %q\n got %q", want, got)
	}
}
