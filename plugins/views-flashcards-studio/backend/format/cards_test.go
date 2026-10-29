package format_test

import (
	"errors"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
)

func TestACardActiveBeforeEverySectionIsNotAtAnyOfThem(t *testing.T) {
	if format.NoSection >= 0 {
		t.Errorf("NoSection is %d, which is a position among the sections", format.NoSection)
	}
}

func TestACardContainsNoValueBelowAFieldItDoesNotHave(t *testing.T) {
	card := format.Card{Values: []format.Value{{Field: "Front", Text: "ant"}}}

	if text, ok := card.Value("Back"); ok || text != "" {
		t.Errorf("Value(Back) = %q, %v; want no value", text, ok)
	}
	if text, ok := card.Value("Front"); !ok || text != "ant" {
		t.Errorf("Value(Front) = %q, %v; want ant", text, ok)
	}
}

func TestADeckContainingNoCardOfAMarkReturnsSo(t *testing.T) {
	deck := format.Deck{Cards: []format.Card{{CardID: "aaaaaaaaaa", Heading: "ant"}}}

	got, err := deck.Card("bbbbbbbbbb")

	if !errors.Is(err, format.ErrNoSuchCard) {
		t.Fatalf("Card(bbbbbbbbbb) error = %v, want ErrNoSuchCard", err)
	}
	if got.Heading != "" || got.CardID != "" {
		t.Errorf("Card(bbbbbbbbbb) = %+v, want no card", got)
	}
}

func TestADeckContainingTwoCardsOfAMarkReturnsSo(t *testing.T) {
	deck := format.Deck{Cards: []format.Card{
		{CardID: cardid.ID("aaaaaaaaaa"), Heading: "ant"},
		{CardID: cardid.ID("aaaaaaaaaa"), Heading: "bee"},
	}}

	if _, err := deck.Card("aaaaaaaaaa"); !errors.Is(err, format.ErrTwoCards) {
		t.Fatalf("Card(aaaaaaaaaa) error = %v, want ErrTwoCards", err)
	}
}

// A card's heading is the first line of its first field, truncated to one line.
// It ends at the first line break and at format.HeadingRunes characters,
// counted in Unicode characters, and leading and trailing spaces are removed.
//
// The value is untrusted input: it is user-written HTML in a file that another
// editor may have written.
func TestAProjectedHeadingIsOneLineOfCharactersTheFieldOpensWith(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.String().Draw(t, "value")
		got := format.Project(value)

		if strings.ContainsAny(got, "\n\r") {
			t.Fatalf("%q projects to %q, which is more than one line", value, got)
		}
		if n := len([]rune(got)); n > format.HeadingRunes {
			t.Fatalf("%q projects to %d characters, and %d is the most a heading is",
				value, n, format.HeadingRunes)
		}
		if got != strings.TrimSpace(got) {
			t.Fatalf("%q projects to %q, which is not trimmed", value, got)
		}
		// A heading holds nothing of its own: every character of it comes off
		// the front of the field's first line.
		line, _, _ := strings.Cut(markdown.Normalise(value), "\n")
		if opens := strings.TrimSpace(line); !strings.HasPrefix(opens, got) {
			t.Fatalf("%q opens with %q and projects to %q, which is not its own",
				value, opens, got)
		}
	})
}

// atom is one piece of a field's first line; isUnbroken reports whether the cut
// may fall inside it.
type atom struct {
	text       string
	isUnbroken bool
}

// drawAtoms generates a first line as a sequence of pieces whose boundaries
// are known to the test: prose the cut may fall anywhere in, and runs it must not
// split: a wikilink, an embed, a run of emphasis, and nested combinations of
// these.
//
// The prose contains no bracket, asterisk, underscore or exclamation mark, so a
// piece the cut may fall inside cannot open a run of its own or close a run an
// earlier piece opened. This lets the test determine every unbroken run without
// repeating the algorithm under test.
func drawAtoms(t *rapid.T) []atom {
	prose := rapid.StringMatching(`[\p{L}\p{N} ,.;:()-]{4,40}`)
	target := rapid.StringMatching(`[\p{L}\p{N} .-]{4,30}`)
	inner := rapid.StringMatching(`[\p{L}\p{N} ,.;:()-]{4,30}`)
	emphasis := rapid.SampledFrom([]string{"*", "**", "***", "_", "__", "___"})

	link := rapid.Custom(func(t *rapid.T) string {
		open := rapid.SampledFrom([]string{"[[", "![["}).Draw(t, "opens")
		return open + target.Draw(t, "target") + "]]"
	})
	run := rapid.Custom(func(t *rapid.T) string {
		cardID := emphasis.Draw(t, "emphasis")
		return cardID + inner.Draw(t, "held") + cardID
	})
	// A link inside a run of emphasis belongs to the link, and the enclosing run
	// is one run; the link brackets may contain any user text, including an embed,
	// and the link ends at its last closing pair.
	nested := rapid.Custom(func(t *rapid.T) string {
		cardID := emphasis.Draw(t, "emphasis")
		return cardID + inner.Draw(t, "before") + link.Draw(t, "link") + cardID
	})
	targetVal := rapid.Custom(func(t *rapid.T) string {
		return "[[" + target.Draw(t, "target") + " ![[" +
			target.Draw(t, "embedded") + "]] " + target.Draw(t, "after") + "]]"
	})

	piece := rapid.Custom(func(t *rapid.T) atom {
		switch rapid.IntRange(0, 4).Draw(t, "kind") {
		case 0:
			return atom{text: link.Draw(t, "link"), isUnbroken: true}
		case 1:
			return atom{text: run.Draw(t, "run"), isUnbroken: true}
		case 2:
			return atom{text: nested.Draw(t, "nested"), isUnbroken: true}
		case 3:
			return atom{text: targetVal.Draw(t, "held"), isUnbroken: true}
		}
		return atom{text: prose.Draw(t, "prose")}
	})
	return rapid.SliceOfN(piece, 2, 24).Draw(t, "line")
}

// The cut never falls inside a wikilink, an embed or a run of emphasis: it
// falls before the whole run that contains the cut position. A heading ending
// inside a link would display markup as the card name.
func TestTheSplitNeverFallsInsideALinkOrARunOfEmphasis(t *testing.T) {
	t.Parallel()
	var trimmed, straddled int
	rapid.Check(t, func(t *rapid.T) {
		atoms := drawAtoms(t)

		var line strings.Builder
		type span struct{ from, to int }
		var runs []span
		pos := 0
		for _, one := range atoms {
			n := len([]rune(one.text))
			if one.isUnbroken {
				runs = append(runs, span{pos, pos + n})
			}
			pos += n
			line.WriteString(one.text)
		}
		value := line.String()
		got := format.Project(value)

		// The spaces at either end are dropped, so the heading is counted from
		// the first character the line has, and so are the runs.
		shift := len([]rune(value)) - len([]rune(strings.TrimLeft(value, " ")))
		for i := range runs {
			runs[i].from -= shift
			runs[i].to -= shift
		}

		whole := len([]rune(strings.TrimSpace(value)))
		if whole > format.HeadingRunes {
			trimmed++
			for _, one := range runs {
				if one.from < format.HeadingRunes && format.HeadingRunes < one.to {
					straddled++
					break
				}
			}
		}

		fell := len([]rune(got))
		for _, one := range runs {
			if one.from < fell && fell < one.to {
				t.Fatalf("%q projects to %q, which stops inside the run at %d:%d",
					value, got, one.from, one.to)
			}
		}
	})
	// The property constrains where a cut may fall, so a generation that never
	// cut a line, or never produced a run spanning the cut position, tests
	// nothing.
	if trimmed < 25 || straddled < 15 {
		t.Fatalf("%d lines were cut and %d of those had a run standing across the cut",
			trimmed, straddled)
	}
}
