package format

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// A line of hashes with no space after them is text, whatever its level.
func TestAHashRunWithNoSpaceIsNotAHeading(t *testing.T) {
	body := "##Llama ^k7m2xq9fzp\n[[Animal]]\n\n# Furry\n\n##Cow\n[[Animal]]\n"
	d := ReadDeck(domain.Note{Body: body})
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
func TestSetCardModelReplacesALinkOnTheLineBelowTheHeading(t *testing.T) {
	raw := "# Furry\n\n## Llama ^k7m2xq9fzp\n[[Animal]]\n\n### Name\n\nLlama\n"
	f := OpenDeckBody(raw)
	if err := f.SetCardModel("k7m2xq9fzp", "Beast"); err != nil {
		t.Fatalf("set the card model: %v", err)
	}
	if want := strings.Replace(raw, "[[Animal]]", "[[Beast]]", 1); f.Body() != want {
		t.Errorf("body = %q, want %q", f.Body(), want)
	}
}

// A problem about a card names the card and no template.
func TestOnCardFilesAProblemAgainstOneCard(t *testing.T) {
	got := OnCard(3, FaultDuplicateCardID, "detail")
	want := Problem{
		Fault: FaultDuplicateCardID, Card: 3, Template: NoPosition, Detail: "detail",
	}
	if got != want {
		t.Errorf("problem = %+v, want %+v", got, want)
	}
}

func createDeckBody(numCards int, linesPerCard int) []byte {
	var b bytes.Buffer
	b.WriteString("---\ntype: deck\n---\n\nDeck description.\n\n")
	for i := range numCards {
		fmt.Fprintf(&b, "# Section %d\n\n", i/10)
		fmt.Fprintf(&b, "## Card %d ^card%06d\n\n", i, i)
		b.WriteString("[[Animal]]\n\n")
		b.WriteString("Card description and notes.\n\n")
		b.WriteString("### Front\n\n")
		b.WriteString("```python\ndef test():\n    return 42\n```\n\n")
		for l := range linesPerCard {
			fmt.Fprintf(&b, "Line of prose number %d with some text.\n", l)
		}
		b.WriteString("\n### Back\n\n")
		b.WriteString("Back of the card explanation.\n\n")
	}
	return b.Bytes()
}

func createProseDenseBody(numLines int) []byte {
	var b bytes.Buffer
	b.WriteString("# Document Title\n\n")
	for i := range numLines {
		switch {
		case i%50 == 0:
			fmt.Fprintf(&b, "## Heading %d\n\n", i)
		case i%20 == 0:
			b.WriteString("```go\nfunc hello() {\n    // fence line\n}\n```\n\n")
		default:
			fmt.Fprintf(&b, "This is line %d with some prose and general text content.\n", i)
		}
	}
	return b.Bytes()
}

func BenchmarkSections(b *testing.B) {
	small := createDeckBody(10, 2)
	large := createDeckBody(1000, 5)
	prose := createProseDenseBody(5000)

	b.Run("DeckSmall", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(small)))
		b.ResetTimer()
		for range b.N {
			_ = sections(small, SectionLevel, FieldLevel)
		}
	})

	b.Run("DeckLarge", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(large)))
		b.ResetTimer()
		for range b.N {
			_ = sections(large, SectionLevel, FieldLevel)
		}
	})

	b.Run("ProseDense", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(prose)))
		b.ResetTimer()
		for range b.N {
			_ = sections(prose, SectionLevel, FieldLevel)
		}
	})
}

func BenchmarkReadDeck(b *testing.B) {
	smallRaw := createDeckBody(10, 2)
	smallNote := domain.Note{FileStat: domain.FileStat{Path: "Deck.md"}, Body: string(smallRaw)}

	largeRaw := createDeckBody(1000, 5)
	largeNote := domain.Note{FileStat: domain.FileStat{Path: "Deck.md"}, Body: string(largeRaw)}

	b.Run("DeckSmall", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(smallRaw)))
		b.ResetTimer()
		for range b.N {
			_ = ReadDeck(smallNote)
		}
	})

	b.Run("DeckLarge", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(largeRaw)))
		b.ResetTimer()
		for range b.N {
			_ = ReadDeck(largeNote)
		}
	})
}
