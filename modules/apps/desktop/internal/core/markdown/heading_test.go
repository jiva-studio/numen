package markdown

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// The key is spliced like any other, so the block around it comes out as it
// went in.
func TestSetTitleWritesTheKeyAndLeavesTheRest(t *testing.T) {
	for name, c := range map[string]struct {
		raw   string
		title string
		want  string
	}{
		"a note with no title key grows one": {
			raw:   "---\nid: 01J8\n---\nbody\n",
			title: "Entropy",
			want:  "---\nid: 01J8\ntitle: Entropy\n---\nbody\n",
		},
		"the title already there is replaced": {
			raw:   "---\ntitle: Old\nid: 01J8\n---\nbody\n",
			title: "Entropy",
			want:  "---\ntitle: Entropy\nid: 01J8\n---\nbody\n",
		},
		"a note with no frontmatter grows one": {
			raw:   "body\n",
			title: "Entropy",
			want:  "---\ntitle: Entropy\n---\nbody\n",
		},
		"a title YAML would read as something else is quoted": {
			raw:   "---\nid: 01J8\n---\nbody\n",
			title: "123",
			want:  "---\nid: 01J8\ntitle: \"123\"\n---\nbody\n",
		},
		"a title carrying a comment marker is quoted": {
			raw:   "---\nid: 01J8\n---\nbody\n",
			title: "# hash",
			want:  "---\nid: 01J8\ntitle: '# hash'\n---\nbody\n",
		},
		"a CRLF note is written with CRLF": {
			raw:   "---\r\nid: 01J8\r\n---\r\nbody\r\n",
			title: "Entropy",
			want:  "---\r\nid: 01J8\r\ntitle: Entropy\r\n---\r\nbody\r\n",
		},
		"a comment above the next key stays with it": {
			raw:   "---\ntitle: Old\n\n# mine\nzebra: 1\n---\nbody\n",
			title: "Entropy",
			want:  "---\ntitle: Entropy\n\n# mine\nzebra: 1\n---\nbody\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(c.raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := d.SetTitle(c.title); err != nil {
				t.Fatalf("set title: %v", err)
			}
			if got := string(d.Bytes()); got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
			if n := Parse(domain.FileRef{Path: "note.md"}, d.Bytes()); n.Title != c.title {
				t.Errorf("the parser reads back %q, not the title written", n.Title)
			}
		})
	}
}

func TestSetHeadingRewritesTheFirstLevelOneHeading(t *testing.T) {
	for name, c := range map[string]struct {
		raw   string
		want  string
		found bool
	}{
		"the heading a note opens with": {
			raw:   "# Old\n\nA measure.\n",
			want:  "# Entropy\n\nA measure.\n",
			found: true,
		},
		"the first of several": {
			raw:   "# Old\n\n# Later\n",
			want:  "# Entropy\n\n# Later\n",
			found: true,
		},
		"a level-one heading below a level-two one": {
			raw:   "## Sub\n\n# Old\n",
			want:  "## Sub\n\n# Entropy\n",
			found: true,
		},
		"a heading closed with hashes": {
			raw:   "# Old ###\n\ntext\n",
			want:  "# Entropy\n\ntext\n",
			found: true,
		},
		"a heading that is the whole note": {
			raw:   "# Old",
			want:  "# Entropy\n",
			found: true,
		},
		"a note whose frontmatter is left alone": {
			raw:   "---\nid: 01J8\n---\n# Old\n",
			want:  "---\nid: 01J8\n---\n# Entropy\n",
			found: true,
		},
		"a CRLF note is written with CRLF": {
			raw:   "# Old\r\n\r\ntext\r\n",
			want:  "# Entropy\r\n\r\ntext\r\n",
			found: true,
		},
		"a heading inside a code fence is an example of one": {
			raw:  "```\n# Old\n```\n",
			want: "```\n# Old\n```\n",
		},
		"a note with no heading at all": {
			raw:  "A measure.\n",
			want: "A measure.\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(c.raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if got := d.SetHeading("Entropy"); got != c.found {
				t.Fatalf("want found=%v, got %v", c.found, got)
			}
			if got := string(d.Bytes()); got != c.want {
				t.Errorf("want %q, got %q", c.want, got)
			}
		})
	}
}

func TestInsertHeadingOpensTheProse(t *testing.T) {
	for name, c := range map[string]struct {
		raw  string
		want string
	}{
		"a note with prose": {
			raw:  "A measure.\n",
			want: "# Entropy\n\nA measure.\n",
		},
		"a note that is empty": {
			raw:  "",
			want: "# Entropy\n",
		},
		"a note whose prose begins after a blank line": {
			raw:  "---\nid: 01J8\n---\n\nA measure.\n",
			want: "---\nid: 01J8\n---\n# Entropy\n\nA measure.\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(c.raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			d.InsertHeading("Entropy")
			if got := string(d.Bytes()); got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A title with a line break in it is one the frontmatter has to spell over
// several lines, and the parser reads back what went in.
func TestATitleOverSeveralLinesSurvivesTheTrip(t *testing.T) {
	d, err := Open([]byte("---\nid: 01J8\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetTitle("one\ntwo"); err != nil {
		t.Fatalf("set title: %v", err)
	}
	got := d.Bytes()
	if !strings.Contains(string(got), "id: 01J8\n") {
		t.Errorf("the key beside it was lost:\n%s", got)
	}
	if n := Parse(domain.FileRef{Path: "note.md"}, got); n.Title != "one\ntwo" {
		t.Errorf("the parser reads back %q", n.Title)
	}
}
