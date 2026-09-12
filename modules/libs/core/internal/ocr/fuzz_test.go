package ocr_test

import (
	"image"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
)

// artifactSeeds are the shapes an artifact arrives in: what a run writes, a
// blank page, a run that stopped part way and said so, prose standing before
// the first page, a mark that never closes, a mark with no newline after it,
// bytes that are no artifact at all, and nothing.
var artifactSeeds = []string{
	"\x0c\x0c\nthe first page\n\n a second region\n\x0c\x0c\nthe second page\n",
	"\x0c\x0c\n\x0c\x0c\nafter a blank page\n",
	"\x0c\x0c\nthe first page\n\x00 stopped at page 1\n",
	"loose prose\n\x0c\x0c\nthe first page\n",
	"\x0c the mark never closes\n",
	"\x0c\x0cno newline after the mark\n",
	"\x00\x00\x00",
	"a\xff\xfeb",
	"\x0c",
	"",
}

// The prose an artifact is read as is the artifact with its bookkeeping taken
// out and nothing else: every byte of it stood in the file, in that order. A
// page begins at an offset in that prose and not past its end, and the pages
// come in the order they were read.
//
// An artifact is a file in a folder inside somebody's vault, written by a run
// that may have stopped at any moment and carried between machines by something
// that knows nothing about it.
func FuzzReadArtifact(f *testing.F) {
	for _, seed := range artifactSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		text, pages := ocr.Read([]byte(raw))
		if !subsequence(text, raw) {
			t.Fatalf("%q was read as %q, which is not what the file says", raw, text)
		}
		at := -1
		for i, page := range pages {
			if page.Offset < 0 || page.Offset > len(text) {
				t.Fatalf("page %d begins at %d in %d bytes of prose",
					i, page.Offset, len(text))
			}
			if page.Offset < at {
				t.Fatalf("page %d begins at %d, before page %d at %d",
					i, page.Offset, i-1, at)
			}
			at = page.Offset
		}
	})
}

// A page written out is read back as itself: as many pages as were written, and
// every part and every box standing at the run of the prose it was taken from.
//
// The artifact is the file, so what is written has to survive being read.
func FuzzWriteArtifact(f *testing.F) {
	f.Add("the first region", "A Heading", "the second page", true, 1)
	f.Add("", "", "", false, 0)
	f.Add(" ", "\n\n", "\t", true, 3)
	f.Add("шабда", "文字", "\xff\xfe", true, 2)
	f.Add(strings.Repeat("word ", 200), "x", "y", false, 0)

	f.Fuzz(func(t *testing.T, first, heading, second string, marked bool, depth int) {
		// A form feed is the page mark and a NUL opens the line a run writes for
		// itself. No recogniser has a character for either, so nothing written
		// here carries one.
		if strings.ContainsAny(first+heading+second, "\x0c\x00") {
			return
		}

		pages := []ocr.Page{{
			Index: 0,
			Size:  image.Point{X: 100, Y: 200},
			Blocks: []ocr.Block{
				{Text: first, Boxes: []ocr.Box{{
					Rect: image.Rect(0, 0, 10, 10),
					Span: domain.Span{From: 0, To: len(first)},
				}}},
				{Text: heading, Heading: marked, Depth: depth},
			},
		}, {
			Index:  1,
			Blocks: []ocr.Block{{Text: second}},
		}}

		raw, boxes, parts := ocr.Write(pages)
		text, read := ocr.Read(raw)
		if len(read) != len(pages) {
			t.Fatalf("%d pages were written and %d read back", len(pages), len(read))
		}
		if marked && heading != "" && len(parts) != 1 {
			t.Fatalf("one heading was written and %d parts came back", len(parts))
		}
		for _, part := range parts {
			if part.Start < 0 || part.Start+part.Length > len(text) {
				t.Fatalf("a part runs from %d for %d in %d bytes of prose",
					part.Start, part.Length, len(text))
			}
			if got := text[part.Start : part.Start+part.Length]; got != heading {
				t.Fatalf("a part of %q stands at %q", heading, got)
			}
		}
		for _, box := range boxes {
			if box.From < 0 || box.To > len(text) {
				t.Fatalf("a box runs from %d to %d in %d bytes of prose",
					box.From, box.To, len(text))
			}
			if got := text[box.From:box.To]; got != first {
				t.Fatalf("a box over %q stands at %q", first, got)
			}
		}
	})
}

// subsequence reports whether every byte of the prose stood in the file, in the
// order the file has them.
func subsequence(prose, raw string) bool {
	at := 0
	for i := range len(prose) {
		for at < len(raw) && raw[at] != prose[i] {
			at++
		}
		if at == len(raw) {
			return false
		}
		at++
	}
	return true
}
