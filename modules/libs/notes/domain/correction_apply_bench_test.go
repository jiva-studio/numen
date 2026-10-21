package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// benchRecognition is a recognition of n printed lines, with the corrections for
// every step-th line, a third of them lengthened and a third shortened.
func benchRecognition(n, step int) (string, []domain.OCRPageStart, []domain.HighlightBox, []domain.OCRPart, []domain.CorrectionLine) {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = strings.Repeat(fmt.Sprintf("word%d ", i), 12)
	}
	var marks []domain.OCRPageStart
	var parts []domain.OCRPart
	boxes := make([]domain.HighlightBox, 0, n)
	at := 0
	for i, line := range lines {
		if i%perPage == 0 {
			marks = append(marks, domain.OCRPageStart{Offset: at})
		}
		if i%10 == 0 {
			parts = append(parts, domain.OCRPart{Start: at, Length: len(line) * 4, Depth: i / 10 % 3})
		}
		boxes = append(boxes, domain.HighlightBox{
			Page:     i / perPage,
			ByteSpan: domain.ByteSpan{From: at, To: at + len(line)},
		})
		at += len(line) + 1
	}
	var placed []domain.CorrectionLine
	for i := 0; i < n; i += step {
		text := lines[i]
		switch (i / step) % 3 {
		case 0:
			text += " extra"
		case 1:
			text = text[:len(text)/2]
		}
		placed = append(placed, domain.CorrectionLine{Number: i, Text: text})
	}
	if len(placed) > 1 {
		placed = append(placed, domain.CorrectionLine{Number: placed[0].Number, Text: "again"})
	}
	return strings.Join(lines, "\n"), marks, boxes, parts, placed
}

var benchCases = []struct {
	name string
	n    int
	step int
}{
	{"200boxes_50pct", 200, 2},
	{"500boxes_20pct", 500, 5},
	{"2000boxes_5pct", 2000, 20},
	{"2000boxes_50pct", 2000, 2},
}

func BenchmarkBoxes(b *testing.B) {
	for _, c := range benchCases {
		_, _, boxes, _, placed := benchRecognition(c.n, c.step)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				domain.GetCorrectedBoxes(boxes, placed)
			}
		})
	}
}

func BenchmarkProse(b *testing.B) {
	for _, c := range benchCases {
		prose, marks, boxes, parts, placed := benchRecognition(c.n, c.step)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				domain.GetCorrectedProse(prose, marks, boxes, parts, placed)
			}
		})
	}
}
