package markdown

import (
	"slices"
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
		"leading lf before crlf": {"\n\r\n", "\n"},
		"trailing cr after crlf": {"\r\n\r", "\n"},
		"mid cr after crlf":      {"\r\n\rx\r\n", "\n"},
		"short crlf":             {"a\r\n", "\r\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := lineEnding([]byte(c.raw)); got != c.want {
				t.Errorf("want %q, got %q", c.want, got)
			}
		})
	}
}

// A note that arrived with CRLF is written back with CRLF, prose and all.
func TestTheProseOfACRLFNoteIsSavedWithCRLF(t *testing.T) {
	d, err := Open([]byte("---\r\nid: 01J8\r\n---\r\nbody\r\n"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := d.SetBody("first\nsecond\n"); err != nil {
		t.Fatalf("set body: %v", err)
	}

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
	if err := d.SetBody("first\nsecond\n"); err != nil {
		t.Fatalf("set body: %v", err)
	}

	if got := string(d.Bytes()); got != raw {
		t.Errorf("the prose was rewritten\n want %q\n  got %q", raw, got)
	}
}

// A carriage return on its own is a line break, and is returned as one.
func TestALoneCarriageReturnIsALineBreak(t *testing.T) {
	if got := Normalize("first\rsecond\r\nthird\n"); got != "first\nsecond\nthird\n" {
		t.Errorf("got %q", got)
	}
	if got := Normalize("nothing to normalize\n"); got != "nothing to normalize\n" {
		t.Errorf("text with no carriage return came back changed: %q", got)
	}
}

// An offset standing where no line begins is the line that begins next, and an
// offset past every line is len(lines).
func TestOffsetLineFindsTheNextLineWhenNoneBeginsThere(t *testing.T) {
	lines := []int{0, 5, 11, 15}
	for name, c := range map[string]struct {
		offset int
		want   int
	}{
		"a line's own start":    {0, 1},
		"mid-line":              {3, 2},
		"the next line's start": {5, 2},
		"mid-line again":        {8, 3},
		"the last line's start": {15, 4},
		"past every line":       {20, 4},
	} {
		t.Run(name, func(t *testing.T) {
			if got := offsetLine(lines, c.offset); got != c.want {
				t.Errorf("offsetLine(%v, %d) = %d, want %d", lines, c.offset, got, c.want)
			}
		})
	}
}

func TestBreakWidthsEndsWithBreakAndLineOffsets(t *testing.T) {
	widthCases := []struct {
		block  []byte
		offset int
		want   int
	}{
		{[]byte("\r"), 0, 1},
		{[]byte("a\r"), 1, 1},
		{[]byte("\n\r"), 1, 1},
		{[]byte("a\r\n"), 1, 2},
		{[]byte("\n"), 0, 1},
		{[]byte{0xC2}, 0, 0},
		{[]byte{'a', 0xC2}, 1, 0},
		{[]byte{0xC2, 0x84}, 0, 0},
		{[]byte{0xC2, 0x85}, 0, 2},
		{[]byte{0xC2, 0x86}, 0, 0},
		{[]byte{0xE2}, 0, 0},
		{[]byte{0xE2, 0x80}, 0, 0},
		{[]byte{'a', 'b', 0xE2, 0x80}, 2, 0},
		{[]byte{0xE2, 0x81, 0xA8}, 0, 0},
		{[]byte{0xE2, 0x80, 0xA7}, 0, 0},
		{[]byte{0xE2, 0x80, 0xA8}, 0, 3},
		{[]byte{0xE2, 0x80, 0xA9}, 0, 3},
		{[]byte{0xE2, 0x80, 0xAA}, 0, 0},
		{[]byte("x"), 0, 0},
	}
	for _, tc := range widthCases {
		if got := breakWidth(tc.block, tc.offset); got != tc.want {
			t.Errorf("breakWidth(%x, %d) = %d, want %d", tc.block, tc.offset, got, tc.want)
		}
	}

	for _, tc := range []struct {
		block []byte
		want  bool
	}{
		{[]byte(""), false},
		{[]byte("a"), false},
		{[]byte("abcd"), false},
		{[]byte("a\n"), true},
		{[]byte("a\r"), true},
		{[]byte("a\r\n"), true},
		{[]byte("a\u0085"), true},
		{[]byte("a\u2028"), true},
		{[]byte("a\u2029"), true},
		{[]byte("\u2028a"), false},
	} {
		if got := endsWithBreak(tc.block); got != tc.want {
			t.Errorf("endsWithBreak(%q) = %v, want %v", tc.block, got, tc.want)
		}
	}

	if got := lineOffsets([]byte("\na")); !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("lineOffsets(\\na) = %v", got)
	}
	if got := lineOffsets([]byte("a\n")); !slices.Equal(got, []int{0, 2}) {
		t.Errorf("lineOffsets(a\\n) = %v", got)
	}
	if got := lineOffsets([]byte("a\nb")); !slices.Equal(got, []int{0, 2, 3}) {
		t.Errorf("lineOffsets(a\\nb) = %v", got)
	}
}
