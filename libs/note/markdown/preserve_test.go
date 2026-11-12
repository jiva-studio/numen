package markdown_test

import (
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
)

// byteOrderMark is what another editor may put in front of a note's first line.
const byteOrderMark = "\uFEFF"

func listTheirBlocks() []string {
	return []string{
		"mine: keep me verbatim\n",
		"# a note to myself\n",
		"zebra: 1\n",
		"read:\n  - one\n  - two\n",
		"aliases: [a, b]\n",
		"empty:\n",
		"quoted: \"a: b\"\n",
		"folded: >\n  a line\n  and another\n",
	}
}

func listIdentifiers() []string {
	return []string{
		"01J8F3K2M9QRSTVWXYZ0123456",
		"7ZZZZZZZZZZZZZZZZZZZZZZZZZ",
		"00000000000000000000000000",
	}
}

func listOurBlocks() []string {
	return []string{
		"id: " + listIdentifiers()[0] + "\n",
		"title: Old\n",
		"type: note\n",
		"fields:\n  - Height\n  - Weight\n",
	}
}

func drawNote(t *rapid.T) (raw string, foreign []string) {
	blocks := append(append([]string{}, listTheirBlocks()...), listOurBlocks()...)
	order := rapid.Permutation(blocks).Draw(t, "keys")
	keys := order[:rapid.IntRange(0, len(order)).Draw(t, "member")]

	var front strings.Builder
	for _, one := range keys {
		front.WriteString(one)
		if !strings.HasPrefix(one, "#") && !contains(listOurBlocks(), one) {
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
	return slices.Contains(all, one)
}

// A note that is read and not written is returned as the bytes it was read as.
// Reading does not rewrite: an unchanged note keeps even its line endings.
//
// The frontmatter is shared with the user, and the application does not own all
// of it.
func TestANoteNeverSavedToReachesOutAsItWentIn(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		raw, _ := drawNote(t)
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
func TestWhatTheApplicationDoesNotOwnIsCachedVerbatim(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		raw, foreign := drawNote(t)
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
				err = doc.SetIdentifier(listIdentifiers()[at%len(listIdentifiers())])
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
			checkForeignKeysKept(t, string(doc.Bytes()), foreign, write)
		}
	})
}

func checkForeignKeysKept(t *rapid.T, note string, foreign []string, wrote string) {
	t.Helper()
	if len(foreign) == 0 {
		return
	}
	// The endings are the note's own, and the identity property is where they
	note = strings.ReplaceAll(strings.TrimPrefix(note, byteOrderMark), "\r\n", "\n")
	front, _, found := strings.Cut(strings.TrimPrefix(note, "---\n"), "\n---\n")
	if !found {
		t.Fatalf("writing %s left no frontmatter block: %q", wrote, note)
	}
	lines := strings.Split(front, "\n")

	idx := 0
	for _, one := range foreign {
		for line := range strings.SplitSeq(strings.TrimSuffix(one, "\n"), "\n") {
			for idx < len(lines) && lines[idx] != line {
				idx++
			}
			if idx == len(lines) {
				t.Fatalf("writing %s lost the line %q, or moved it:\n%s", wrote, line, note)
			}
			idx++
		}
	}
}

// What is written to an owned key is what is read back from it, by the writer
// and by the reader of a whole note alike. A title is what the note is called
// afterwards, and an identifier is the one the note carries.
//
// The frontmatter is where the application's fields live.
func TestAnOwnedKeyReadsBackAsItWasSaved(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		raw, _ := drawNote(t)
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
		identifier := rapid.SampledFrom(listIdentifiers()).Draw(t, "id")

		if err := doc.SetTitle(title); err != nil {
			return
		}
		if err := doc.SetIdentifier(identifier); err != nil {
			return
		}
		if err := doc.SetList("fields", names); err != nil {
			return
		}

		if got, target := doc.Title(); got != strings.TrimSpace(title) || !target {
			t.Fatalf("a note written as %q is called %q", title, got)
		}
		if got, targetVal := doc.Identifier(); got != identifier || !targetVal {
			t.Fatalf("a note stamped %q carries %q", identifier, got)
		}
		// A key written no names at all is a key the note does not carry.
		got, targetItem := doc.List("fields")
		if targetItem != (len(names) > 0) || !equal(got, names) {
			t.Fatalf("a note written the fields %v holds %v, %v", names, got, targetItem)
		}

		// And the reader of a whole note reads what the writer wrote.
		n := markdown.Parse(domain.FileStat{Path: "notes/Old.md"}, doc.Bytes())
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
	for idx := range got {
		if got[idx] != want[idx] {
			return false
		}
	}
	return true
}
