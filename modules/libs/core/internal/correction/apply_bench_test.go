package correction_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/correction"
	"github.com/jiva-studio/numen/modules/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
)

// benchReading is a reading of n printed lines, with the corrections for
// every step-th line, a third of them lengthened and a third shortened.
func benchReading(n, step int) (string, []ocr.PageStart, []highlight.Box, []ocr.Part, []correction.Line) {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = strings.Repeat(fmt.Sprintf("word%d ", i), 12)
	}
	var marks []ocr.PageStart
	var parts []ocr.Part
	boxes := make([]highlight.Box, 0, n)
	at := 0
	for i, line := range lines {
		if i%perPage == 0 {
			marks = append(marks, ocr.PageStart{Offset: at})
		}
		if i%10 == 0 {
			parts = append(parts, ocr.Part{Start: at, Length: len(line) * 4, Depth: i / 10 % 3})
		}
		boxes = append(boxes, highlight.Box{
			Page:     i / perPage,
			ByteSpan: domain.ByteSpan{From: at, To: at + len(line)},
		})
		at += len(line) + 1
	}
	var put []correction.Line
	for i := 0; i < n; i += step {
		text := lines[i]
		switch (i / step) % 3 {
		case 0:
			text += " extra"
		case 1:
			text = text[:len(text)/2]
		}
		put = append(put, correction.Line{Number: i, Text: text})
	}
	if len(put) > 1 {
		put = append(put, correction.Line{Number: put[0].Number, Text: "again"})
	}
	return strings.Join(lines, "\n"), marks, boxes, parts, put
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
		_, _, boxes, _, put := benchReading(c.n, c.step)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				correction.Boxes(boxes, put)
			}
		})
	}
}

func BenchmarkProse(b *testing.B) {
	for _, c := range benchCases {
		prose, marks, boxes, parts, put := benchReading(c.n, c.step)
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				correction.Prose(prose, marks, boxes, parts, put)
			}
		})
	}
}
