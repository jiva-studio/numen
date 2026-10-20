package format_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/format"
)

const twoFieldCard = "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\n1.8 m\n"

// A value set to nothing leaves its heading with the blank line a heading is
// followed by, and the fields after it keep their places.
func TestSetValueToNothingKeepsTheFieldsAfterIt(t *testing.T) {
	f := format.OpenDeckBody(twoFieldCard)
	if err := f.SetValue("k7m2xq9fzp", "Name", ""); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\n### Height\n\n1.8 m\n"
	if got := f.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// The last field of the file set to nothing ends the file on its heading.
func TestSetTheLastValueToNothingEndsTheFileOnItsHeading(t *testing.T) {
	f := format.OpenDeckBody(twoFieldCard)
	if err := f.SetValue("k7m2xq9fzp", "Height", ""); err != nil {
		t.Fatalf("set: %v", err)
	}
	want := "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n"
	if got := f.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// A card written after text that does not end its line stands two breaks below
// it.
func TestAddCardAfterTextWithoutALineBreak(t *testing.T) {
	f := format.OpenDeckBody("Notes on animals")
	if err := f.AddCard(format.Card{Heading: "Alpaca"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	want := "Notes on animals\n\n## Alpaca\n"
	if got := f.Body(); got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// What a section's lead holds belongs to the section, so a deck that ends on
// that lead has only the break that ends the file.
func TestADeckEndingOnASectionLeadHasOnlyTheFinalBreakAsTail(t *testing.T) {
	deck := format.ReadDeck(note(t, "# Pets\n\nSome lead.\n"))
	if len(deck.Sections) != 1 || deck.Sections[0].Preamble != "Some lead." {
		t.Fatalf("sections = %+v", deck.Sections)
	}
	if deck.Tail != "\n" {
		t.Errorf("tail = %q, want the final break", deck.Tail)
	}
}

// A line that closes no pair of brackets after a link that does is not a link,
// and the link before it still holds the cut off.
func TestProjectKeepsALinkWhenALaterPairIsNeverClosed(t *testing.T) {
	head := strings.Repeat("a", format.HeadingRunes-3)
	got := format.Project(head + " [[xy]] [[never closed and more")
	if got != head {
		t.Errorf("project = %q, want %q", got, head)
	}
}

// A file that carries an identifier is left as it is, and one that carries none
// is given it.
func TestWriteIdentifier(t *testing.T) {
	t.Run("a file carrying none", func(t *testing.T) {
		f, err := format.OpenDeck([]byte("---\ntitle: Animals\n---\n" + twoFieldCard))
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		wrote, err := f.WriteIdentifier("abc123")
		if err != nil || !wrote {
			t.Fatalf("write = %v, %v, want true, nil", wrote, err)
		}
		doc, err := markdown.Open(f.Bytes())
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		if id, ok := doc.Identifier(); !ok || id != "abc123" {
			t.Errorf("identifier = %q, %v", id, ok)
		}
	})

	t.Run("a file carrying one", func(t *testing.T) {
		raw := "---\nid: first\n---\n" + twoFieldCard
		f, err := format.OpenDeck([]byte(raw))
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		wrote, err := f.WriteIdentifier("second")
		if err != nil || wrote {
			t.Fatalf("write = %v, %v, want false, nil", wrote, err)
		}
		if got := string(f.Bytes()); got != raw {
			t.Errorf("file changed: %q", got)
		}
	})

	t.Run("a frontmatter that cannot be written", func(t *testing.T) {
		f, err := format.OpenDeck([]byte("---\n{title: Animals}\n---\n" + twoFieldCard))
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if _, err := f.WriteIdentifier("abc123"); !errors.Is(err, markdown.ErrInline) {
			t.Errorf("err = %v, want ErrInline", err)
		}
	})
}
