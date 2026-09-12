package markdown

import (
	"testing"
	"unicode/utf8"
)

// The offsets are into the text as it was given, so what a caller replaces is
// the bytes the person wrote.
func TestASpanIsFoundWhereItStands(t *testing.T) {
	text := "the aggressor is named"
	at, plainly := Where(text, "aggressor")
	if plainly {
		t.Error("a span that stands exactly was read plainly")
	}
	if len(at) != 1 {
		t.Fatalf("found %d places, wanted one", len(at))
	}
	if got := text[at[0].From:at[0].To]; got != "aggressor" {
		t.Errorf("the offsets cover %q", got)
	}
}

// Two of the same span are two answers, and a caller may refuse them.
func TestASpanWrittenTwiceIsFoundTwice(t *testing.T) {
	at, _ := Where("a foe, and another foe", "foe")
	if len(at) != 2 {
		t.Fatalf("found %d places, wanted two", len(at))
	}
	if at[0].From != 2 || at[1].From != 19 {
		t.Errorf("found at %d and %d", at[0].From, at[1].From)
	}
}

// A span inside a span is stepped over, so nothing overlaps.
func TestPlacesFoundDoNotOverlap(t *testing.T) {
	at, _ := Where("aaaa", "aa")
	if len(at) != 2 {
		t.Fatalf("found %d places, wanted two", len(at))
	}
	if at[0].To != at[1].From {
		t.Errorf("the first ends at %d and the second begins at %d", at[0].To, at[1].From)
	}
}

// Punctuation a person's editor writes and a program rarely reproduces is read
// as the mark it stands for, and the reading is reported.
func TestPunctuationIsReadAsTheMarkItStandsFor(t *testing.T) {
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
			at, plainly := Where(c.text, c.wanted)
			if len(at) != 1 {
				t.Fatalf("found %d places, wanted one", len(at))
			}
			if !plainly {
				t.Error("the reading was not reported")
			}
			if at[0].From != 0 || at[0].To != len(c.text) {
				t.Errorf("the offsets cover %q, wanted the whole of it", c.text[at[0].From:at[0].To])
			}
		})
	}
}

// The offsets are into the text as it stands, whichever reading found them, so
// a replacement leaves the bytes around it as the person wrote them.
func TestOffsetsAreIntoTheTextAsItStands(t *testing.T) {
	text := "before «дa» after"
	at, _ := Where(text, "\"дa\"")
	if len(at) != 1 {
		t.Fatalf("found %d places, wanted one", len(at))
	}
	if got := text[at[0].From:at[0].To]; got != "«дa»" {
		t.Errorf("the offsets cover %q", got)
	}
	if left := text[:at[0].From]; left != "before " {
		t.Errorf("what stands before is %q", left)
	}
}

// A run of spacing inside a line is read as one space, so prose reflowed by an
// editor is still found.
func TestARunOfSpacingInsideALineIsOneSpace(t *testing.T) {
	at, plainly := Where("the  foe\tadvances", "the foe advances")
	if len(at) != 1 {
		t.Fatalf("found %d places, wanted one", len(at))
	}
	if !plainly {
		t.Error("the reading was not reported")
	}
}

// A break between lines is spacing of its own. A caller asking for one line
// does not get two.
func TestABreakBetweenLinesIsNotReadAsASpace(t *testing.T) {
	if at, _ := Where("the foe\nadvances", "the foe advances"); at != nil {
		t.Errorf("found %d places across a break", len(at))
	}
}

// A span that stands exactly is never reported where it only nearly stands.
func TestWhatStandsExactlyIsAnsweredBeforeWhatNearlyDoes(t *testing.T) {
	text := "he said \"yes\" and he said “yes”"
	at, plainly := Where(text, "\"yes\"")
	if plainly {
		t.Error("a span standing exactly was read plainly")
	}
	if len(at) != 1 {
		t.Fatalf("found %d places, wanted one", len(at))
	}
	if at[0].From != 8 {
		t.Errorf("found at %d, wanted the one that stands exactly", at[0].From)
	}
}

// Nothing is what an empty span finds. Every place would be the alternative.
func TestAnEmptySpanFindsNothing(t *testing.T) {
	if at, _ := Where("some prose", ""); at != nil {
		t.Errorf("found %d places for nothing", len(at))
	}
}

// A span nobody wrote is not found, however it is read.
func TestASpanThatIsNotThereIsNotFound(t *testing.T) {
	if at, _ := Where("the aggressor is named", "the poisoner"); at != nil {
		t.Errorf("found %d places", len(at))
	}
}

// A note rewritten whole still names the sentence that changed, so what is
// drawn is the change and not the note.
func TestOnlyWhatDiffersIsAnswered(t *testing.T) {
	was := "# Title\n\nA hedgehog is named.\n\nAnd nothing else.\n"
	now := "# Title\n\nAn axe is named.\n\nAnd nothing else.\n"

	at, insert := Differs(was, now)
	if was[at.From:at.To] != "A hedgehog" {
		t.Errorf("the span is %q", was[at.From:at.To])
	}
	if insert != "An axe" {
		t.Errorf("what goes in is %q", insert)
	}
}

// Two texts that are the same name no span at all.
func TestTextThatDidNotChangeAnswersAnEmptySpan(t *testing.T) {
	at, insert := Differs("the same", "the same")
	if at.From != at.To || insert != "" {
		t.Errorf("answered %d..%d with %q", at.From, at.To, insert)
	}
}

// A span never begins or ends inside a rune, whatever the two texts share.
func TestASpanNeverSplitsARune(t *testing.T) {
	at, insert := Differs("сказал «да» сразу", "сказал «нет» сразу")
	if !utf8.ValidString(insert) {
		t.Errorf("what goes in is not text: %q", insert)
	}
	if !utf8.RuneStart("сказал «да» сразу"[at.From]) {
		t.Errorf("the span begins inside a rune, at %d", at.From)
	}
}

// A client counts text its own way, and prose that is not ASCII is where the
// two partings show.
func TestAnOffsetIsCountedTheWayAClientCountsText(t *testing.T) {
	for name, c := range map[string]struct {
		text string
		at   int
		want int
	}{
		"ascii":              {"one two", 4, 4},
		"cyrillic":           {"сказал да", len("сказал "), 7},
		"beyond the basics":  {"a 𝔘 b", len("a 𝔘"), 4},
		"the end of it":      {"сказал", len("сказал"), 6},
		"the start of it":    {"сказал", 0, 0},
		"past the end of it": {"сказал", 999, 6},
	} {
		t.Run(name, func(t *testing.T) {
			if got := CountUTF16(c.text, c.at); got != c.want {
				t.Errorf("counted %d, wanted %d", got, c.want)
			}
		})
	}
}
