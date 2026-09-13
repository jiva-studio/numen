package chunking_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
)

// cutSeeds are texts a cut has to survive: prose, a book that puts a chapter on
// one line, a text of no letters at all, one of a single word longer than any
// limit, one written in another script, and the empty text.
var cutSeeds = []string{
	"The first sentence. The second sentence.\n\nA second paragraph, longer than the first one is.\n",
	strings.Repeat("word ", 400),
	"..... ,,,,, ;;;;; !!!!!\n",
	strings.Repeat("a", 4000),
	"शब्द शब्द शब्द\nदूसरी पंक्ति\n",
	"мысль, и ещё одна мысль\n",
	"a\x00b\xff\xfe c",
	string(rune(0xFEFF)) + "word word\r\nword\r\n",
	"",
}

// Every chunk of a cut is a stretch of the text it was cut from: inside it, of
// no negative length, and cut at a character and never through one. A small
// chunk stands under a large chunk that reaches its middle.
//
// The text is a stranger's: a source's text is whatever the file held.
func FuzzCut(f *testing.F) {
	for _, seed := range cutSeeds {
		f.Add(seed, 8, 2, 3, 1, 40, 3)
	}
	f.Add("one two three four five", 2, 1, 1, 0, 4, 0)
	f.Add("one two three four five", chunking.Whole, 0, 2, 1, 1000, 2)

	f.Fuzz(func(t *testing.T,
		text string, large, largeOverlap, small, smallOverlap, limit, part int,
	) {
		sizes := chunking.Sizes{
			Large: large, Small: small,
			LargeOverlap: largeOverlap, SmallOverlap: smallOverlap,
			Limit: limit,
		}
		// A part begins where its title was found in the text, so it stands at a
		// character. A part offset inside one is not a text a caller can hand
		// over, and the chunks cut from it begin inside a character too.
		var parts []chunking.PartStart
		if len(text) > 0 {
			at := part % len(text)
			for at > 0 && !utf8.RuneStart(text[at]) {
				at--
			}
			parts = []chunking.PartStart{
				{Title: "one", Offset: 0},
				{Title: "two", Offset: at},
			}
		}

		// A text that is not UTF-8 has no characters to cut at, and the bounds
		// are all there is to hold a chunk to.
		cuts := utf8.ValidString(text)
		for _, one := range chunking.Cut(text, parts, sizes, chunking.Legibility{}) {
			assertChunkInText(t, text, one, cuts, "a large chunk")
			for _, held := range one.Small {
				assertChunkInText(t, text, held, cuts, "a small chunk")
				if middle := held.Start + held.Length/2; middle < one.Start ||
					middle >= one.Start+max(one.Length, 1) {
					t.Fatalf("a small chunk at %d+%d stands under a large one at %d+%d",
						held.Start, held.Length, one.Start, one.Length)
				}
				if held.Length == 0 {
					continue
				}
				if held.Location != one.Location && one.Location != "" {
					t.Fatalf("a small chunk of %q stands under a large chunk of %q",
						held.Location, one.Location)
				}
			}
		}
	})
}

// assertChunkInText fails unless a chunk is a stretch of the text, cut at a
// character.
func assertChunkInText(t *testing.T, text string, c chunking.Chunk, cuts bool, what string) {
	t.Helper()
	switch {
	case c.Length < 0 || c.Start < 0 || c.Start+c.Length > len(text):
		t.Fatalf("%s runs from %d for %d in a text of %d bytes",
			what, c.Start, c.Length, len(text))
	case !cuts:
	case c.Start < len(text) && !utf8.RuneStart(text[c.Start]):
		t.Fatalf("%s at %d begins inside a character", what, c.Start)
	case c.Start+c.Length < len(text) && !utf8.RuneStart(text[c.Start+c.Length]):
		t.Fatalf("%s ending at %d ends inside a character", what, c.Start+c.Length)
	}
}
