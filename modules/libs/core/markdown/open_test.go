package markdown_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// TestANoteWithNoFrontmatterIsWrittenTo. Three dashes below the first line are
// a rule in the prose. The whole file is body, and a write reaches it.
func TestANoteWithNoFrontmatterIsWrittenTo(t *testing.T) {
	raw := "# Title\n\n---\n\nnot frontmatter\n"

	d, err := markdown.Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if d.Body() != raw {
		t.Fatalf("the body opened as %q", d.Body())
	}
	if err := d.SpliceBody(0, len("# Title"), "# Another"); err != nil {
		t.Fatalf("splice: %v", err)
	}
	if got, want := string(d.Bytes()), "# Another\n\n---\n\nnot frontmatter\n"; got != want {
		t.Errorf("wrote %q, want %q", got, want)
	}
}

// TestASpliceOutsideTheBodyIsRefused. Bounds that name no run of the prose are
// answered with an error.
func TestASpliceOutsideTheBodyIsRefused(t *testing.T) {
	d, err := markdown.Open([]byte("---\nid: 01J\n---\nbody\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	for _, one := range []struct {
		name       string
		start, end int
	}{
		{"beginning before the body", -1, 2},
		{"ending before it begins", 3, 1},
		{"reaching past the end", 0, 999},
	} {
		if err := d.SpliceBody(one.start, one.end, "x"); err == nil {
			t.Errorf("%s: %d:%d was allowed", one.name, one.start, one.end)
		}
	}
}

// TestANoteMadeHereEndsWithALineEnding. What a person starts a note with is
// theirs, and the file it lands in ends the way a text file ends.
func TestANoteMadeHereEndsWithALineEnding(t *testing.T) {
	for _, body := range []string{"# Note", "# Note\n", ""} {
		if got := string(markdown.Create("01J", body)); !strings.HasSuffix(got, "\n") {
			t.Errorf("a note started with %q was made as %q", body, got)
		}
	}
}
