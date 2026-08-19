package markdown

import (
	"strings"
	"unicode/utf8"
)

// Stretch is a run of prose, as byte offsets into the text it was found in.
// Where a client is told about a run it is told in the units a client counts
// in, which is domain.Span; nothing crosses that boundary unconverted.
type Stretch struct {
	From int
	To   int
}

// Where is every place `wanted` stands in `text`.
//
// Three readings are tried and the first that finds anything answers: the text
// as it stands, then punctuation and spacing read as the marks they stand for,
// then a run of spacing inside a line read as one space. A stretch that stands
// exactly is therefore never reported where it only nearly stands.
//
// The offsets are into `text` as it was given, whichever reading found them, so
// what is replaced is the bytes the person wrote.
//
// Plainly says the stretch was found only once punctuation or spacing were
// allowed to differ, which is worth saying out loud to whoever asked.
func Where(text, wanted string) (at []Stretch, plainly bool) {
	if wanted == "" {
		return nil, false
	}
	if found := standing(text, wanted); len(found) > 0 {
		return found, false
	}
	for _, read := range []func(string) reading{plain, loose} {
		held, sought := read(text), read(wanted)
		if sought.text == "" {
			continue
		}
		found := standing(held.text, sought.text)
		if len(found) == 0 {
			continue
		}
		at = make([]Stretch, 0, len(found))
		for _, span := range found {
			at = append(at, Stretch{From: held.at[span.From], To: held.at[span.To]})
		}
		return at, true
	}
	return nil, false
}

// standing is every place `wanted` stands in `text`. A place found is stepped
// over, so two reported stretches never overlap.
func standing(text, wanted string) []Stretch {
	var at []Stretch
	for from := 0; from <= len(text); {
		next := strings.Index(text[from:], wanted)
		if next < 0 {
			return at
		}
		at = append(at, Stretch{From: from + next, To: from + next + len(wanted)})
		from += next + len(wanted)
	}
	return at
}

// A reading is text as one of the readings has it, and where each of its bytes
// came from. There is one more offset than there are bytes, so the end of a
// stretch reads back as well as its beginning.
type reading struct {
	text string
	at   []int
}

// plain reads typographic punctuation and spacing as the marks they stand for.
func plain(text string) reading {
	var out strings.Builder
	at := make([]int, 0, len(text)+1)
	for from, r := range text {
		before := out.Len()
		out.WriteRune(plainly(r))
		for range out.Len() - before {
			at = append(at, from)
		}
	}
	return reading{text: out.String(), at: append(at, len(text))}
}

// loose reads a run of spacing inside a line as one space. A break between
// lines is spacing of its own and is left where it is.
func loose(text string) reading {
	var out strings.Builder
	at := make([]int, 0, len(text)+1)
	spacing := false
	for from, r := range text {
		mark := plainly(r)
		switch {
		case mark == '\n':
			spacing = false
		case mark == ' ' || mark == '\t':
			if spacing {
				continue
			}
			mark, spacing = ' ', true
		default:
			spacing = false
		}
		before := out.Len()
		out.WriteRune(mark)
		for range out.Len() - before {
			at = append(at, from)
		}
	}
	return reading{text: out.String(), at: append(at, len(text))}
}

// plainly is the mark a rune stands for, or the rune itself. Dashes, quotes and
// the spaces that are not the space bar are what a person's editor puts in
// their prose and a program writing about that prose rarely reproduces.
func plainly(r rune) rune {
	switch {
	// Hyphens and dashes, U+2010 to U+2015, and the minus sign U+2212.
	case r >= '‐' && r <= '―', r == '−':
		return '-'
	// Single quotes, U+2018 to U+201B, and the angled pair U+2039 and U+203A.
	case r >= '‘' && r <= '‛', r == '‹', r == '›':
		return '\''
	// Double quotes, U+201C to U+201F, and the angled pair U+00AB and U+00BB.
	case r >= '“' && r <= '‟', r == '«', r == '»':
		return '"'
	// Spaces that are not the space bar: U+00A0, U+2002 to U+200A, U+202F,
	// U+205F and U+3000.
	case r == ' ', r >= ' ' && r <= ' ', r == ' ',
		r == ' ', r == '　':
		return ' '
	}
	return r
}

// Differs is the stretch of `was` that `now` does not have, and the text that
// stands there instead.
//
// What the two share at either end is left out, so replacing one whole note's
// prose with another names the sentence that changed. The stretch is widened to
// whole words, and two texts that are the same name no stretch at all.
func Differs(was, now string) (Stretch, string) {
	if was == now {
		return Stretch{From: len(was), To: len(was)}, ""
	}

	head := 0
	for head < len(was) && head < len(now) && was[head] == now[head] {
		head++
	}
	for head > 0 && !opens(was, head) {
		head--
	}

	most := min(len(was), len(now)) - head
	tail := 0
	for tail < most && was[len(was)-tail-1] == now[len(now)-tail-1] {
		tail++
	}
	for tail > 0 && !closes(was, len(was)-tail) {
		tail--
	}

	return Stretch{From: head, To: len(was) - tail}, now[head : len(now)-tail]
}

// opens reports whether a stretch may begin at `at`: at the start of the text,
// or where a rune begins and spacing stands before it.
func opens(text string, at int) bool {
	if at <= 0 {
		return true
	}
	if at < len(text) && !utf8.RuneStart(text[at]) {
		return false
	}
	return spacing(text[at-1])
}

// closes reports whether a stretch may end at `at`: at the end of the text, or
// where spacing stands.
func closes(text string, at int) bool {
	if at >= len(text) {
		return true
	}
	if !utf8.RuneStart(text[at]) {
		return false
	}
	return spacing(text[at])
}

func spacing(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// Counted is `at`, a byte offset into text, as a client counts text: in UTF-16
// code units.
//
// A note is read by something that counts its own way, and a stretch named in
// bytes lands somewhere else in prose that is not ASCII.
func Counted(text string, at int) int {
	if at > len(text) {
		at = len(text)
	}
	units := 0
	for _, r := range text[:at] {
		units++
		if r > 0xffff {
			units++
		}
	}
	return units
}
