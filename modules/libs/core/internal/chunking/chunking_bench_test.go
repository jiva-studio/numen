package chunking

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func BenchmarkIsLegible(b *testing.B) {
	thresholds := Legibility{}.resolve()
	smallText := line(50)
	largeText := line(200)
	multilingualText := strings.Repeat(latin+" "+cyrillic+" "+sanskrit+" ", 5)
	noisyText := strings.Repeat(noise+" ", 10)

	b.Run("small", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = isLegible(smallText, thresholds)
		}
	})

	b.Run("large", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = isLegible(largeText, thresholds)
		}
	})

	b.Run("multilingual", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = isLegible(multilingualText, thresholds)
		}
	})

	b.Run("noise", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = isLegible(noisyText, thresholds)
		}
	})
}

func BenchmarkCut(b *testing.B) {
	text := line(1000)
	parts := []domain.PartStart{
		{Title: "Part 1", Offset: 0},
		{Title: "Part 2", Offset: len(line(250)) + 1},
		{Title: "Part 3", Offset: len(line(500)) + 2},
		{Title: "Part 4", Offset: len(line(750)) + 3},
	}
	reads := Legibility{}

	b.Run("default", func(b *testing.B) {
		sizes := Sizes{}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = Cut(text, parts, sizes, reads)
		}
	})

	b.Run("whole", func(b *testing.B) {
		sizes := Sizes{Large: Whole}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = Cut(text, parts, sizes, reads)
		}
	})

	b.Run("long", func(b *testing.B) {
		longText := line(10000)
		sizes := Sizes{}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = Cut(longText, nil, sizes, reads)
		}
	})
}
