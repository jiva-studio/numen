package format

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func createDeckBody(numCards int, linesPerCard int) []byte {
	var b bytes.Buffer
	b.WriteString("---\ntype: deck\n---\n\nDeck description.\n\n")
	for i := range numCards {
		b.WriteString(fmt.Sprintf("# Section %d\n\n", i/10))
		b.WriteString(fmt.Sprintf("## Card %d ^card%06d\n\n", i, i))
		b.WriteString("[[Animal]]\n\n")
		b.WriteString("Card description and notes.\n\n")
		b.WriteString("### Front\n\n")
		b.WriteString("```python\ndef test():\n    return 42\n```\n\n")
		for l := range linesPerCard {
			b.WriteString(fmt.Sprintf("Line of prose number %d with some text.\n", l))
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
			b.WriteString(fmt.Sprintf("## Heading %d\n\n", i))
		case i%20 == 0:
			b.WriteString("```go\nfunc hello() {\n    // fence line\n}\n```\n\n")
		default:
			b.WriteString(fmt.Sprintf("This is line %d with some prose and general text content.\n", i))
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
	smallNote := domain.Note{Fingerprint: domain.Fingerprint{Path: "Deck.md"}, Body: string(smallRaw)}

	largeRaw := createDeckBody(1000, 5)
	largeNote := domain.Note{Fingerprint: domain.Fingerprint{Path: "Deck.md"}, Body: string(largeRaw)}

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
