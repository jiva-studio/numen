package cards_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
)

// A mark stands last, one space after the heading's text, and is a mark only at
// that length and in that alphabet. Anything else at the end of a heading is
// heading text.
func TestReadHeading(t *testing.T) {
	for name, one := range map[string]struct {
		heading, text string
		mark          cardid.CardID
	}{
		"a text and a mark": {
			"Compost, what is it made of ^k7m2xq9fzp", "Compost, what is it made of", "k7m2xq9fzp",
		},
		"a mark and nothing else": {"^k7m2xq9fzp", "", "k7m2xq9fzp"},
		"nothing at all":          {"", "", ""},
		"no mark":                 {"Compost, what is it made of", "Compost, what is it made of", ""},
		"too few characters":      {"Compost ^k7m2xq9fz", "Compost ^k7m2xq9fz", ""},
		"too many characters":     {"Compost ^k7m2xq9fzpp", "Compost ^k7m2xq9fzpp", ""},
		"outside the alphabet":    {"Compost ^K7M2XQ9FZP", "Compost ^K7M2XQ9FZP", ""},
		"no caret":                {"Compost k7m2xq9fzp", "Compost k7m2xq9fzp", ""},
		"no space in front":       {"Compost^k7m2xq9fzp", "Compost^k7m2xq9fzp", ""},
		"not last on the line":    {"Compost ^k7m2xq9fzp is a heap", "Compost ^k7m2xq9fzp is a heap", ""},
		"a caret and nothing":     {"Compost ^", "Compost ^", ""},
		"a text of one space":     {"Compost  ^k7m2xq9fzp", "Compost ", "k7m2xq9fzp"},
		"a text that is a mark":   {"^k7m2xq9fzp ^zpqrstvwxy", "^k7m2xq9fzp", "zpqrstvwxy"},
	} {
		t.Run(name, func(t *testing.T) {
			text, mark := cards.ReadHeading(one.heading)
			if text != one.text || mark != one.mark {
				t.Errorf("read %q = %q, %q, want %q, %q", one.heading, text, mark, one.text, one.mark)
			}
			if got := cards.WriteHeading(text, mark); got != one.heading {
				t.Errorf("written back = %q, want %q", got, one.heading)
			}
		})
	}
}

// A card whose first field is empty stands under a heading of its mark alone,
// and a heading is the text alone until the card is written.
func TestWriteHeading(t *testing.T) {
	for name, one := range map[string]struct {
		text string
		mark cardid.CardID
		want string
	}{
		"both":         {"Compost", "k7m2xq9fzp", "Compost ^k7m2xq9fzp"},
		"no text":      {"", "k7m2xq9fzp", "^k7m2xq9fzp"},
		"no mark":      {"Compost", "", "Compost"},
		"neither":      {"", "", ""},
		"beyond latin": {"компост", "k7m2xq9fzp", "компост ^k7m2xq9fzp"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := cards.WriteHeading(one.text, one.mark); got != one.want {
				t.Errorf("write = %q, want %q", got, one.want)
			}
		})
	}
}

// The heading is the first line of the first field, cut to fit one line: it
// stops at the first break and at cards.HeadingRunes characters.
func TestProject(t *testing.T) {
	long := strings.Repeat("a", cards.HeadingRunes+40)
	upTo := func(n int) string { return strings.Repeat("a", n) }

	for name, one := range map[string]struct{ value, want string }{
		"one line":            {"Compost, what is it made of", "Compost, what is it made of"},
		"beyond latin":        {"компост, из чего он", "компост, из чего он"},
		"two lines":           {"Compost, what is it made of\nand what it is for", "Compost, what is it made of"},
		"carriage returns":    {"Compost\r\nand more", "Compost"},
		"a lone carriage":     {"Compost\rand more", "Compost"},
		"nothing":             {"", ""},
		"spaces alone":        {"   ", ""},
		"spaces around":       {"  Compost  ", "Compost"},
		"a blank first line":  {"\nCompost", ""},
		"too long":            {long, upTo(cards.HeadingRunes)},
		"a break before then": {upTo(10) + "\n" + long, upTo(10)},

		// The cut never falls inside one of these: it falls before the whole of
		// whichever it lands in.
		"inside a wikilink": {
			upTo(cards.HeadingRunes-5) + " [[Compost heap]] and on", upTo(cards.HeadingRunes - 5),
		},
		"inside an embed": {
			upTo(cards.HeadingRunes-5) + " ![[llama.png]] and on", upTo(cards.HeadingRunes - 5),
		},
		"inside a run of emphasis": {
			upTo(cards.HeadingRunes-5) + " **very heavy** and on", upTo(cards.HeadingRunes - 5),
		},
		"inside emphasis of one star": {
			upTo(cards.HeadingRunes-5) + " *very heavy* and on", upTo(cards.HeadingRunes - 5),
		},
		"inside emphasis of an underscore": {
			upTo(cards.HeadingRunes-5) + " __very heavy__ and on", upTo(cards.HeadingRunes - 5),
		},
		// A wikilink the cut falls after is text like any other.
		"after a wikilink": {
			upTo(cards.HeadingRunes-20) + " [[Compost heap]] " + long,
			upTo(cards.HeadingRunes-20) + " [[Compost heap]] " + upTo(2),
		},
		// A star with no partner opens no run of emphasis.
		"a lone star": {
			upTo(cards.HeadingRunes-5) + " *very heavy and on", upTo(cards.HeadingRunes-5) + " *ver",
		},

		// A link inside a run of emphasis is the link's, and the run around it
		// is one run.
		"inside emphasis holding a link": {
			upTo(100) + " *see [[a very long target]] now*", upTo(100),
		},
		"inside emphasis holding an embed": {
			upTo(100) + " **the [[compost]] one made of leaves**", upTo(100),
		},
		// A link's brackets hold whatever a person wrote, brackets included, and
		// the link closes at its own last one.
		"inside a wikilink holding an embed": {
			upTo(100) + " [[a ![[img.png]] b]] tail", upTo(100),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := cards.Project(one.value)
			if got != one.want {
				t.Errorf("project %q\n want %q\n  got %q", one.value, one.want, got)
			}
			if n := len([]rune(got)); n > cards.HeadingRunes {
				t.Errorf("the heading is %d characters, and %d is the most one is", n, cards.HeadingRunes)
			}
		})
	}
}
