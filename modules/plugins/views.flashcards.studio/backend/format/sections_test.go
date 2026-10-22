package format_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
)

// A line of hashes with no space after them is text, whatever its level.
func TestAHashRunWithNoSpaceIsNotAHeading(t *testing.T) {
	body := "##Llama ^k7m2xq9fzp\n[[Animal]]\n\n# Furry\n\n##Cow\n[[Animal]]\n"
	d := format.ReadDeck(domain.Note{Body: body})
	if len(d.Cards) != 0 {
		t.Errorf("cards = %+v, want none", d.Cards)
	}
	if len(d.Sections) != 1 || d.Sections[0].Name != "Furry" {
		t.Errorf("sections = %+v, want the one named Furry", d.Sections)
	}
	if want := "##Cow\n[[Animal]]"; d.Sections[0].Preamble != want {
		t.Errorf("preamble = %q, want %q", d.Sections[0].Preamble, want)
	}
}

// A wikilink standing on the first line under its card's heading is replaced
// in place, and the heading above it stays.
func TestSetStencilReplacesALinkOnTheLineBelowTheHeading(t *testing.T) {
	raw := "# Furry\n\n## Llama ^k7m2xq9fzp\n[[Animal]]\n\n### Name\n\nLlama\n"
	f := format.OpenDeckBody(raw)
	if err := f.SetStencil("k7m2xq9fzp", "Beast"); err != nil {
		t.Fatalf("set the stencil: %v", err)
	}
	if want := strings.Replace(raw, "[[Animal]]", "[[Beast]]", 1); f.Body() != want {
		t.Errorf("body = %q, want %q", f.Body(), want)
	}
}

// A problem about a card names the card and no face.
func TestOnCardFilesAProblemAgainstOneCard(t *testing.T) {
	got := format.OnCard(3, format.FaultTwoMarks, "detail")
	want := format.Problem{
		Fault: format.FaultTwoMarks, Card: 3, Face: format.NoPosition, Detail: "detail",
	}
	if got != want {
		t.Errorf("problem = %+v, want %+v", got, want)
	}
}
