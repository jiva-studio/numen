package markdown_test

import (
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// byteOrderMark is what another editor may put in front of a note's first line.
var byteOrderMark = string(rune(0xFEFF))

// theirs are frontmatter blocks this application owns no key of: a person's own
// keys, a comment they wrote to themselves, a list, a flow sequence, a key with
// nothing under it, and one whose value would be read as a mapping if it were
// not quoted.
var theirs = []string{
	"mine: keep me verbatim\n",
	"# a note to myself\n",
	"zebra: 1\n",
	"read:\n  - one\n  - two\n",
	"aliases: [a, b]\n",
	"empty:\n",
	"quoted: \"a: b\"\n",
	"folded: >\n  a line\n  and another\n",
}

// ours are the keys this application owns, as a note that has been written here
// already carries them.
// identifiers are ULIDs, which is what an owned `id` holds.
var identifiers = []string{
	"01J8F3K2M9QRSTVWXYZ0123456",
	"7ZZZZZZZZZZZZZZZZZZZZZZZZZ",
	"00000000000000000000000000",
}

var ours = []string{
	"id: " + identifiers[0] + "\n",
	"title: Old\n",
	"type: note\n",
	"fields:\n  - Height\n  - Weight\n",
}

// noted generates a note: a frontmatter block of some of those keys in some
// order, and prose under it.
func noted(t *rapid.T) (raw string, foreign []string) {
	blocks := append(append([]string{}, theirs...), ours...)
	order := rapid.Permutation(blocks).Draw(t, "keys")
	keys := order[:rapid.IntRange(0, len(order)).Draw(t, "held")]

	var front strings.Builder
	for _, one := range keys {
		front.WriteString(one)
		if !strings.HasPrefix(one, "#") && !contains(ours, one) {
			foreign = append(foreign, one)
		}
	}
	body := rapid.SampledFrom([]string{
		"", "A measure of disorder.\n", "# Old\n\nprose\n",
		"text with a [[wikilink]] in it\n", "---\nnot frontmatter\n",
	}).Draw(t, "body")

	// The line endings another editor wrote, and the byte order mark another
	// one puts in front of them.
	raw = "---\n" + front.String() + "---\n" + body
	if rapid.Bool().Draw(t, "crlf") {
		raw = strings.ReplaceAll(raw, "\n", "\r\n")
	}
	if rapid.Bool().Draw(t, "mark") {
		raw = byteOrderMark + raw
	}
	return raw, foreign
}

func contains(all []string, one string) bool {
	for _, held := range all {
		if held == one {
			return true
		}
	}
	return false
}

// A note read and not written comes out as the bytes it went in as. Reading is
// not rewriting: a note nobody changed is untouched down to its line endings.
//
// The frontmatter is shared with the person, and the application does not own
// the whole of it.
func TestANoteNobodyWroteToComesOutAsItWentIn(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		raw, _ := noted(t)
		doc, err := markdown.Open([]byte(raw))
		if err != nil {
			return
		}
		if got := string(doc.Bytes()); got != raw {
			t.Fatalf("a note read and not written came out changed\n was %q\n now %q", raw, got)
		}
	})
}

// A key the application does not own is preserved verbatim, order included,
// however many owned keys are written over it and in whatever order.
//
// Keys the application does not own are preserved verbatim, order included.
func TestWhatTheApplicationDoesNotOwnIsKeptVerbatim(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		raw, foreign := noted(t)
		doc, err := markdown.Open([]byte(raw))
		if err != nil {
			return
		}

		writes := rapid.SliceOfN(rapid.SampledFrom([]string{
			"title", "id", "fields", "body",
		}), 1, 6).Draw(t, "writes")
		for at, write := range writes {
			switch write {
			case "title":
				err = doc.SetTitle(rapid.SampledFrom([]string{
					"Entropy", "TCP/IP", "a: b", "  spaced  ", "многословие",
				}).Draw(t, "title"))
			case "id":
				err = doc.SetIdentifier(identifiers[at%len(identifiers)])
			case "fields":
				err = doc.SetList("fields", rapid.SliceOfN(rapid.SampledFrom([]string{
					"Height", "Weight", "Shoulder height",
				}), 0, 3).Draw(t, "names"))
			case "body":
				err = doc.SetBody(rapid.SampledFrom([]string{
					"", "new prose\n", "# A heading\n\nand a sentence\n",
				}).Draw(t, "prose"))
			}
			if err != nil {
				return
			}
			kept(t, string(doc.Bytes()), foreign, write)
		}
	})
}

// kept fails unless every line of every key the application does not own still
// stands in the note's frontmatter, unchanged and in the order it stood in.
func kept(t *rapid.T, note string, foreign []string, wrote string) {
	t.Helper()
	if len(foreign) == 0 {
		return
	}
	// The endings are the note's own, and the identity property is where they
	// are held to. Here it is the lines and their order.
	note = strings.ReplaceAll(strings.TrimPrefix(note, byteOrderMark), "\r\n", "\n")
	front, _, found := strings.Cut(strings.TrimPrefix(note, "---\n"), "\n---\n")
	if !found {
		t.Fatalf("writing %s left no frontmatter block: %q", wrote, note)
	}
	lines := strings.Split(front, "\n")

	at := 0
	for _, one := range foreign {
		for _, line := range strings.Split(strings.TrimSuffix(one, "\n"), "\n") {
			for at < len(lines) && lines[at] != line {
				at++
			}
			if at == len(lines) {
				t.Fatalf("writing %s lost the line %q, or moved it:\n%s", wrote, line, note)
			}
			at++
		}
	}
}

// What is written to an owned key is what is read back from it, by the writer
// and by the reader of a whole note alike. A title is what the note is called
// afterwards, and an identifier is the one the note carries.
//
// The frontmatter is where the application's fields live.
func TestAnOwnedKeyReadsBackAsItWasWritten(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		raw, _ := noted(t)
		doc, err := markdown.Open([]byte(raw))
		if err != nil {
			return
		}
		title := rapid.SampledFrom([]string{
			"Entropy", "TCP/IP", "a: b", "  spaced  ", "многословие", "- dashed",
			"#hash", "yes", "123", "[bracketed]",
		}).Draw(t, "title")
		names := rapid.SliceOfN(rapid.SampledFrom([]string{
			"Height", "Weight", "a: b", "über",
		}), 0, 3).Draw(t, "names")
		identifier := rapid.SampledFrom(identifiers).Draw(t, "id")

		if err := doc.SetTitle(title); err != nil {
			return
		}
		if err := doc.SetIdentifier(identifier); err != nil {
			return
		}
		if err := doc.SetList("fields", names); err != nil {
			return
		}

		if got, held := doc.Title(); got != strings.TrimSpace(title) || !held {
			t.Fatalf("a note written as %q is called %q", title, got)
		}
		if got, held := doc.Identifier(); got != identifier || !held {
			t.Fatalf("a note stamped %q carries %q", identifier, got)
		}
		// A key written no names at all is a key the note does not carry.
		got, held := doc.List("fields")
		if held != (len(names) > 0) || !equal(got, names) {
			t.Fatalf("a note written the fields %v holds %v, %v", names, got, held)
		}

		// And the reader of a whole note reads what the writer wrote.
		n := markdown.Parse(domain.Fingerprint{Path: "notes/Old.md"}, doc.Bytes())
		if n.Title != strings.TrimSpace(title) {
			t.Fatalf("a note written as %q is read as %q", title, n.Title)
		}
		if n.ID != identifier {
			t.Fatalf("a note stamped %q is read as %q, with %v", identifier, n.ID, n.Problems)
		}
	})
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return len(got) == 0 && len(want) == 0
	}
	for at := range got {
		if got[at] != want[at] {
			return false
		}
	}
	return true
}
