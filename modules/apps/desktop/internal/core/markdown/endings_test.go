package markdown

import (
	"strings"
	"testing"
)

// What separates the lines is decided by every break in the file, so a note an
// agent appended to keeps the breaks it already has.
func TestALineEndingIsWhatEveryBreakUses(t *testing.T) {
	for name, c := range map[string]struct {
		raw  string
		want string
	}{
		"newlines":               {"---\nid: 01J8\n---\nbody\n", "\n"},
		"carriage returns":       {"---\r\nid: 01J8\r\n---\r\nbody\r\n", "\r\n"},
		"appended after CRLF":    {"---\r\nid: 01J8\r\n---\r\nfirst\nsecond\n", "\n"},
		"appended after newline": {"---\nid: 01J8\n---\nfirst\r\nsecond\r\n", "\n"},
		"a lone carriage return": {"first\rsecond\n", "\n"},
		"one at the very end":    {"first\n", "\n"},
		"nothing to break at":    {"one long line", "\n"},
		"empty":                  {"", "\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := lineEnding([]byte(c.raw)); got != c.want {
				t.Errorf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A note that arrived with CRLF is written back with CRLF, prose and all.
func TestTheProseOfACRLFNoteIsWrittenWithCRLF(t *testing.T) {
	d, err := Open([]byte("---\r\nid: 01J8\r\n---\r\nbody\r\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	d.SetBody("first\nsecond\n")

	got := string(d.Bytes())
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare newline was written into a CRLF note:\n%q", got)
	}
	if want := "---\r\nid: 01J8\r\n---\r\nfirst\r\nsecond\r\n"; got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

// A file with breaks of both kinds is written with bare newlines, so the half
// already written that way is left alone. A note rewritten end to end is cut
// again and embedded again, line for line.
func TestAMixedNoteIsNotRewrittenEndToEnd(t *testing.T) {
	raw := "---\r\nid: 01J8\n---\nfirst\nsecond\n"
	d, err := Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	d.SetBody("first\nsecond\n")

	if got := string(d.Bytes()); got != raw {
		t.Errorf("the prose was rewritten\n want %q\n  got %q", raw, got)
	}
}

// A carriage return on its own is a line break, and comes back as one.
func TestALoneCarriageReturnIsALineBreak(t *testing.T) {
	if got := Normalised("first\rsecond\r\nthird\n"); got != "first\nsecond\nthird\n" {
		t.Errorf("got %q", got)
	}
	if got := Normalised("nothing to normalise\n"); got != "nothing to normalise\n" {
		t.Errorf("text with no carriage return came back changed: %q", got)
	}
}
