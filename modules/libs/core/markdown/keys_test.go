package markdown_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// block is the awkward frontmatter every key written here is written into: a
// key nobody owns above, a key nobody owns below, a comment somebody wrote to
// themselves, and a list in the middle.
const block = "---\n" +
	"id: 01J8F3K2M9QRSTVWXYZ012\n" +
	"# a note to myself\n" +
	"mine: keep me verbatim\n" +
	"fields:\n" +
	"  - Height\n" +
	"  - Weight\n" +
	"zebra: 1\n" +
	"---\n" +
	"\n" +
	"## Recognise\n"

func openDocument(t *testing.T, raw string) *markdown.Document {
	t.Helper()
	doc, err := markdown.Open([]byte(raw))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return doc
}

// A key is written by replacing the lines it occupies, so every other line of
// the block comes out of a write as the bytes it went in as.
func TestWritingOneKeyLeavesTheRestOfTheBlockAsItWas(t *testing.T) {
	doc := openDocument(t, block)
	if err := doc.SetList("fields", []string{"Height", "Shoulder height", "Weight"}); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := string(doc.Bytes())
	want := "---\n" +
		"id: 01J8F3K2M9QRSTVWXYZ012\n" +
		"# a note to myself\n" +
		"mine: keep me verbatim\n" +
		"fields:\n" +
		"  - Height\n" +
		"  - Shoulder height\n" +
		"  - Weight\n" +
		"zebra: 1\n" +
		"---\n" +
		"\n" +
		"## Recognise\n"
	if got != want {
		t.Errorf("wrote\n%q\nwant\n%q", got, want)
	}
}

// The names come back in the order they stand in, and an entry that is not text
// is not a name.
func TestAListIsRead(t *testing.T) {
	doc := openDocument(t, "---\nfields:\n  - Height\n  - 12\n  - Weight\n---\n")
	got, found := doc.List("fields")
	if !found || !slices.Equal(got, []string{"Height", "Weight"}) {
		t.Errorf("list = %v, %v", got, found)
	}

	doc = openDocument(t, "---\nfields: Height\n---\n")
	if got, found := doc.List("fields"); found || got != nil {
		t.Errorf("a key holding no list came back as one: %v", got)
	}
}

// How somebody spells their own frontmatter is theirs, so a name that survives
// a write survives it spelled as it was.
func TestANameKeepsTheWayItWasWritten(t *testing.T) {
	doc := openDocument(t, "---\nfields:\n  - \"Height\"\n  - 'Weight'\n---\n")
	if err := doc.SetList("fields", []string{"Height", "Weight", "Life span"}); err != nil {
		t.Fatalf("set: %v", err)
	}

	want := "---\nfields:\n  - \"Height\"\n  - 'Weight'\n  - Life span\n---\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("wrote\n%q\nwant\n%q", got, want)
	}
}

// A key the block does not carry yet is added to the end of it, and no names
// takes it away again.
func TestAKeyArrivesAndLeaves(t *testing.T) {
	doc := openDocument(t, "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n\n# Entropy\n")
	if err := doc.SetList("fields", []string{"Height"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := "---\nid: 01J8F3K2M9QRSTVWXYZ012\nfields:\n  - Height\n---\n\n# Entropy\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("wrote\n%q\nwant\n%q", got, want)
	}

	if err := doc.SetList("fields", nil); err != nil {
		t.Fatalf("unset: %v", err)
	}
	want = "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n\n# Entropy\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("wrote\n%q\nwant\n%q", got, want)
	}
}

// A scalar key is written the same way, and an empty value takes it away.
func TestAScalarKeyIsWritten(t *testing.T) {
	doc := openDocument(t, "---\nid: 01J8F3K2M9QRSTVWXYZ012\nmine: keep me verbatim\n---\n")
	if err := doc.SetScalar("type", "deck"); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := "---\nid: 01J8F3K2M9QRSTVWXYZ012\nmine: keep me verbatim\ntype: deck\n---\n"
	if got := string(doc.Bytes()); got != want {
		t.Errorf("wrote\n%q\nwant\n%q", got, want)
	}

	if err := doc.SetScalar("type", ""); err != nil {
		t.Fatalf("unset: %v", err)
	}
	if got := string(doc.Bytes()); strings.Contains(got, "type") {
		t.Errorf("the key stayed: %q", got)
	}
}

// A file written with \r\n keeps it, line for line.
func TestAKeyTakesTheFilesOwnLineEnding(t *testing.T) {
	doc := openDocument(t, strings.ReplaceAll(block, "\n", "\r\n"))
	if err := doc.SetList("fields", []string{"Height", "Life span"}); err != nil {
		t.Fatalf("set: %v", err)
	}

	got := string(doc.Bytes())
	if !strings.Contains(got, "fields:\r\n  - Height\r\n  - Life span\r\nzebra") {
		t.Errorf("the file's own line ending was not kept: %q", got)
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare break was written into a \\r\\n file: %q", got)
	}
}

// A block written on one line is a block where the span of any key is the span
// of all of them, so nothing in it is changed.
func TestAnInlineBlockIsRefused(t *testing.T) {
	doc := openDocument(t, "---\n{type: stencil, fields: [Height]}\n---\n")
	if err := doc.SetList("fields", []string{"Height", "Weight"}); err != markdown.ErrInline {
		t.Errorf("set = %v, want it refused", err)
	}
	if err := doc.SetScalar("type", "deck"); err != markdown.ErrInline {
		t.Errorf("set = %v, want it refused", err)
	}
}
