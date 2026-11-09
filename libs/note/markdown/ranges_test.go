package markdown

import (
	"slices"
	"testing"
	"unicode/utf8"
)

// The offsets index the text as given, so a replacement by the caller affects
// the bytes the user wrote.
func TestASpanIsFoundWhereItExists(t *testing.T) {
	text := "the aggressor is named"
	spans, plainly := FindRanges(text, "aggressor")
	if plainly {
		t.Error("a span that stands exactly was read plainly")
	}
	if len(spans) != 1 {
		t.Fatalf("found %d places, wanted one", len(spans))
	}
	if got := text[spans[0].Start:spans[0].End]; got != "aggressor" {
		t.Errorf("the offsets cover %q", got)
	}
}

// Two of the same span are two answers, and a caller may refuse them.
func TestASpanSavedTwiceIsFoundTwice(t *testing.T) {
	spans, _ := FindRanges("a foe, and another foe", "foe")
	if len(spans) != 2 {
		t.Fatalf("found %d places, wanted two", len(spans))
	}
	if spans[0].Start != 2 || spans[1].Start != 19 {
		t.Errorf("found at %d and %d", spans[0].Start, spans[1].Start)
	}
}

// A span inside a span is stepped over, so nothing overlaps.
func TestPlacesFoundDoNotOverlap(t *testing.T) {
	spans, _ := FindRanges("aaaa", "aa")
	if len(spans) != 2 {
		t.Fatalf("found %d places, wanted two", len(spans))
	}
	if spans[0].End != spans[1].Start {
		t.Errorf("the first ends at %d and the second begins at %d", spans[0].End, spans[1].Start)
	}
}

// Punctuation produced by user editors and rarely by programs is matched as the
// mark it represents, and the normalization is reported.
func TestPunctuationIsReadAsTheMarkItExistsFor(t *testing.T) {
	for name, c := range map[string]struct{ text, wanted string }{
		"angled double quotes": {"он сказал «да» сразу", "он сказал \"да\" сразу"},
		"turned double quotes": {"he said “yes” at once", "he said \"yes\" at once"},
		"turned single quote":  {"it is the chair’s key", "it is the chair's key"},
		"an em dash":           {"the hedge — cut — grew back", "the hedge - cut - grew back"},
		"an en dash":           {"chapters 3–12 of it", "chapters 3-12 of it"},
		"a minus sign":         {"a value of −5 here", "a value of -5 here"},
		"a non-breaking space": {"Seed shed opens", "Seed shed opens"},
		"an en quad":           {"Seed shed opens", "Seed shed opens"},
		"an ideographic space": {"Seed　shed opens", "Seed shed opens"},
	} {
		t.Run(name, func(t *testing.T) {
			spans, plainly := FindRanges(c.text, c.wanted)
			if len(spans) != 1 {
				t.Fatalf("found %d places, wanted one", len(spans))
			}
			if !plainly {
				t.Error("the reading was not reported")
			}
			if spans[0].Start != 0 || spans[0].End != len(c.text) {
				t.Errorf("the offsets cover %q, wanted the whole of it", c.text[spans[0].Start:spans[0].End])
			}
		})
	}
}

// The offsets index the current text regardless of which matching pass found
// them, so a replacement leaves the surrounding bytes as the user wrote them.
func TestOffsetsAreIntoTheTextAsItExists(t *testing.T) {
	text := "before «дa» after"
	spans, _ := FindRanges(text, "\"дa\"")
	if len(spans) != 1 {
		t.Fatalf("found %d places, wanted one", len(spans))
	}
	if got := text[spans[0].Start:spans[0].End]; got != "«дa»" {
		t.Errorf("the offsets cover %q", got)
	}
	if remaining := text[:spans[0].Start]; remaining != "before " {
		t.Errorf("what is before is %q", remaining)
	}
}

// editor is still found.
func TestARunOfSpacingInsideALineIsOneSpace(t *testing.T) {
	spans, plainly := FindRanges("the  foe\tadvances", "the foe advances")
	if len(spans) != 1 {
		t.Fatalf("found %d places, wanted one", len(spans))
	}
	if !plainly {
		t.Error("the reading was not reported")
	}
}

// A break between lines is spacing of its own. A caller asking for one line
// does not get two.
func TestABreakBetweenLinesIsNotReadAsASpace(t *testing.T) {
	if spans, _ := FindRanges("the foe\nadvances", "the foe advances"); spans != nil {
		t.Errorf("found %d places across a break", len(spans))
	}
}

// An exact span takes precedence: an approximate span is never reported in its place.
func TestFindRanges_ExactMatchPrecedesApproximate(t *testing.T) {
	text := "he said \"yes\" and he said “yes”"
	spans, plainly := FindRanges(text, "\"yes\"")
	if plainly {
		t.Error("a span standing exactly was read plainly")
	}
	if len(spans) != 1 {
		t.Fatalf("found %d places, wanted one", len(spans))
	}
	if spans[0].Start != 8 {
		t.Errorf("found at %d, wanted the one that stands exactly", spans[0].Start)
	}
}

// Nothing is what an empty span finds. Every place would be the alternative.
func TestAnEmptySpanFindsNothing(t *testing.T) {
	if spans, approx := FindRanges("some prose", ""); spans != nil || approx {
		t.Errorf("found %d places for nothing (approx=%v)", len(spans), approx)
	}
}

// A span that does not appear in the text is not found by any matching mode.
func TestASpanThatIsNotThereIsNotFound(t *testing.T) {
	if spans, approx := FindRanges("the aggressor is named", "the poisoner"); spans != nil || approx {
		t.Errorf("found %d places (approx=%v)", len(spans), approx)
	}
}

// A note rewritten whole still names the sentence that changed, so what is
// drawn is the change and not the note.
func TestDiff_ReturnsOnlyChangedSpan(t *testing.T) {
	was := "# Title\n\nA hedgehog is named.\n\nAnd nothing else.\n"
	now := "# Title\n\nAn axe is named.\n\nAnd nothing else.\n"

	span, insert := Diff(was, now)
	if was[span.Start:span.End] != "A hedgehog" {
		t.Errorf("the span is %q", was[span.Start:span.End])
	}
	if insert != "An axe" {
		t.Errorf("what goes in is %q", insert)
	}
}

// Two texts that are the same name no span at all.
func TestTextThatDidNotChangeAnswersAnEmptySpan(t *testing.T) {
	span, insert := Diff("the same", "the same")
	if span != (Range{Start: 8, End: 8}) || insert != "" {
		t.Errorf("answered %d..%d with %q", span.Start, span.End, insert)
	}
}

// A span never begins or ends inside a rune, whatever the two texts share.
func TestASpanNeverSplitsARune(t *testing.T) {
	span, insert := Diff("сказал «да» сразу", "сказал «нет» сразу")
	if !utf8.ValidString(insert) {
		t.Errorf("what goes in is not text: %q", insert)
	}
	if !utf8.RuneStart("сказал «да» сразу"[span.Start]) {
		t.Errorf("the span begins inside a rune, at %d", span.Start)
	}
}

// A client counts text its own way, and prose that is not ASCII is where the
// two partings show.
func TestAnOffsetIsCountedTheWayAClientCountsText(t *testing.T) {
	for name, c := range map[string]struct {
		text         string
		targetOffset int
		want         int
	}{
		"ascii":              {"one two", 4, 4},
		"cyrillic":           {"сказал да", len("сказал "), 7},
		"beyond the basics":  {"a 𝔘 b", len("a 𝔘"), 4},
		"the end of it":      {"сказал", len("сказал"), 6},
		"the start of it":    {"сказал", 0, 0},
		"past the end of it": {"сказал", 999, 6},
	} {
		t.Run(name, func(t *testing.T) {
			if got := CountUTF16(c.text, c.targetOffset); got != c.want {
				t.Errorf("counted %d, wanted %d", got, c.want)
			}
		})
	}
}

func TestRangesFoldingAndDiffBoundaries(t *testing.T) {
	// foldPunctuation preserves exact spacing count while normalizing dash.
	spans, approx := FindRanges("a  —  b and a — b", "a  -  b")
	if !approx || len(spans) != 1 {
		t.Errorf("expected 1 punctuation match, got %v (approx=%v)", spans, approx)
	}

	if got := foldSpacing("a \n b").text; got != "a \n b" {
		t.Errorf("foldSpacing across newline = %q", got)
	}
	if got := foldSpacing("a b  c").text; got != "a b c" {
		t.Errorf("foldSpacing multiple words = %q", got)
	}
	mapped := foldSpacing("a é b")
	if mapped.text != "a é b" || !slices.Equal(mapped.offsets, []int{0, 1, 2, 2, 4, 5, 6}) {
		t.Errorf("foldSpacing(a é b) = %+v", mapped)
	}

	for r, want := range map[rune]rune{
		'\u200f': '\u200f', '\u2010': '-', '\u2015': '-', '\u2016': '\u2016', '−': '-',
		'\u2017': '\u2017', '\u2018': '\'', '\u201b': '\'', '\u201c': '"', '\u201f': '"',
		'\u2020': '\u2020', '‹': '\'', '›': '\'', '«': '"', '»': '"',
		'\u00a0': ' ', '\u2001': '\u2001', '\u2002': ' ', '\u200a': ' ', '\u200b': '\u200b',
		'\u202f': ' ', '\u205f': ' ', '\u3000': ' ',
	} {
		if got := getPlainRune(r); got != want {
			t.Errorf("getPlainRune(%U) = %q, want %q", r, got, want)
		}
	}

	// Diff boundary cases.
	if sp, ins := Diff("a", "b"); sp != (Range{Start: 0, End: 1}) || ins != "b" {
		t.Errorf("Diff(a, b) = %+v, %q", sp, ins)
	}
	if sp, ins := Diff("hello ", "hello world"); sp != (Range{Start: 6, End: 6}) || ins != "world" {
		t.Errorf("Diff(hello , hello world) = %+v, %q", sp, ins)
	}
	if sp, ins := Diff("hello world", "hello "); sp != (Range{Start: 6, End: 11}) || ins != "" {
		t.Errorf("Diff(hello world, hello ) = %+v, %q", sp, ins)
	}
	if sp, ins := Diff("abc def", "axc def"); sp != (Range{Start: 0, End: 3}) || ins != "axc" {
		t.Errorf("Diff(abc def, axc def) = %+v, %q", sp, ins)
	}
	if sp, ins := Diff("word", "wXrd"); sp != (Range{Start: 0, End: 4}) || ins != "wXrd" {
		t.Errorf("Diff(word, wXrd) = %+v, %q", sp, ins)
	}
	if sp, ins := Diff("abc ", "abd\n"); sp != (Range{Start: 0, End: 4}) || ins != "abd\n" {
		t.Errorf("Diff(abc , abd\\n) = %+v, %q", sp, ins)
	}
	if sp, ins := Diff("a  ", "a "); sp != (Range{Start: 2, End: 3}) || ins != "" {
		t.Errorf("Diff(a  , a ) = %+v, %q", sp, ins)
	}

	if !canSpanBeginAt("a", -1) || !canSpanBeginAt("a", 0) || canSpanBeginAt("a", 1) || !canSpanBeginAt("a ", 2) || canSpanBeginAt("é", 1) {
		t.Error("canSpanBeginAt boundary failure")
	}
	if !canSpanEndAt("a", 1) || !canSpanEndAt("a", 2) || canSpanEndAt("a", 0) || !canSpanEndAt("a b", 1) || canSpanEndAt("é", 1) {
		t.Error("canSpanEndAt boundary failure")
	}
	if !isSpace(' ') || !isSpace('\t') || !isSpace('\n') || !isSpace('\r') || isSpace('a') {
		t.Error("isSpace boundary failure")
	}
	if got := CountUTF16("\uffff\U00010000", 7); got != 3 {
		t.Errorf("CountUTF16(\\uffff\\U00010000) = %d, want 3", got)
	}
}
