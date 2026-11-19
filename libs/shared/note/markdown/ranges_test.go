package markdown

import (
	"testing"
	"unicode/utf8"
)

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

func TestDiffBoundaries(t *testing.T) {

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
