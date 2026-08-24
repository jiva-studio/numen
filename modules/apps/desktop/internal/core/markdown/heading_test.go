package markdown

import (
	"errors"
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
		"a heading that is the whole note keeps the break it had": {
			raw:   "# Old",
			want:  "# Entropy",
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
		"a note of both breaks keeps both": {
			raw:   "# Old\nplain\r\nreturned\n",
			want:  "# Entropy\nplain\r\nreturned\n",
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
			found, err := d.SetHeading("Entropy")
			if err != nil {
				t.Fatalf("set heading: %v", err)
			}
			if found != c.found {
				t.Fatalf("want found=%v, got %v", c.found, found)
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
		"a note whose prose keeps the breaks it had": {
			raw:  "plain\r\nreturned\n",
			want: "# Entropy\n\nplain\r\nreturned\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			d, err := Open([]byte(c.raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := d.InsertHeading("Entropy"); err != nil {
				t.Fatalf("insert heading: %v", err)
			}
			if got := string(d.Bytes()); got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A heading says what a reader takes out of it, and a text a reader takes
// something else out of is refused by both writers rather than written.
func TestAHeadingIsNotWrittenWhereItWouldBeReadAsSomethingElse(t *testing.T) {
	for name, text := range map[string]string{
		"a trailing hash closes the heading":  "C#",
		"a run of trailing hashes":            "Old ###",
		"a hash with nothing after it":        "Draft #",
		"a line break is a second line":       "one\ntwo",
		"a line break makes a second heading": "one\n# two",
		"nothing at all":                      "",
		"space on the ends":                   "  spaced  ",
	} {
		t.Run(name, func(t *testing.T) {
			if Headable(text) {
				t.Fatalf("a heading was said to say %q", text)
			}

			const raw = "# Old\n\nA measure.\n"
			d, err := Open([]byte(raw))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			found, err := d.SetHeading(text)
			if !errors.Is(err, ErrNotAHeading) {
				t.Errorf("want ErrNotAHeading, got %v", err)
			}
			if found {
				t.Error("the heading was said to be written")
			}
			if got := string(d.Bytes()); got != raw {
				t.Errorf("the note was written:\n%q", got)
			}

			e, err := Open([]byte("A measure.\n"))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := e.InsertHeading(text); !errors.Is(err, ErrNotAHeading) {
				t.Errorf("want ErrNotAHeading, got %v", err)
			}
			if got := string(e.Bytes()); got != "A measure.\n" {
				t.Errorf("the note was written:\n%q", got)
			}
		})
	}
}

// What a heading is written with is what a reader takes back out of it.
func TestAHeadingSaysWhatItWasWritten(t *testing.T) {
	for _, text := range []string{
		"Entropy", "Issue #42", "C sharp", "a\tb", "#tag", "```", "Notes [[draft]]",
	} {
		if !Headable(text) {
			t.Errorf("a heading was said not to say %q", text)
			continue
		}
		d, err := Open([]byte("# Old\n"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.SetHeading(text); err != nil {
			t.Fatalf("set heading %q: %v", text, err)
		}
		if got := Parse(domain.FileRef{Path: "note.md"}, d.Bytes()).Title; got != text {
			t.Errorf("the note is shown as %q, and was named %q", got, text)
		}
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
