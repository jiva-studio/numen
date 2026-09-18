package format_test

import (
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// A card's heading is the first line of its first field, cut to fit one line.
// It stops at the first break and at format.HeadingRunes characters, counted as
// a person counts them, and the spaces at either end are dropped.
//
// The value is a stranger's: it is HTML a person wrote, in a file another
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

// atom is one piece of a field's first line, and unbroken says whether the cut
// may fall inside it.
type atom struct {
	text       string
	isUnbroken bool
}

// drawAtoms generates a first line as a run of pieces the test knows the bounds
// of: prose the cut may fall anywhere in, and the runs it may not — a wikilink,
// an embed, a run of emphasis, and the shapes that stand one inside another.
//
// The prose carries no bracket, no star, no underscore and no exclamation mark,
// so a piece the cut may fall inside can neither open a run of its own nor
// close one an earlier piece opened. That is what lets the test say where every
// unbroken run is without working it out the way the code under test does.
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
		mark := emphasis.Draw(t, "emphasis")
		return mark + inner.Draw(t, "held") + mark
	})
	// A link inside a run of emphasis is the link's, and the run around it is
	// one run; a link's brackets hold whatever a person wrote, an embed
	// included, and the link closes at its own last pair.
	nested := rapid.Custom(func(t *rapid.T) string {
		mark := emphasis.Draw(t, "emphasis")
		return mark + inner.Draw(t, "before") + link.Draw(t, "link") + mark
	})
	held := rapid.Custom(func(t *rapid.T) string {
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
			return atom{text: held.Draw(t, "held"), isUnbroken: true}
		}
		return atom{text: prose.Draw(t, "prose")}
	})
	return rapid.SliceOfN(piece, 2, 24).Draw(t, "line")
}

// The cut never falls inside a wikilink, an embed or a run of emphasis: it
// falls in front of the whole of whichever it lands in. A heading ending inside
// a link is markup a person is shown as the name of their card.
func TestTheCutNeverFallsInsideALinkOrARunOfEmphasis(t *testing.T) {
	t.Parallel()
	var cut, straddled int
	rapid.Check(t, func(t *rapid.T) {
		atoms := drawAtoms(t)

		var line strings.Builder
		type span struct{ from, to int }
		var runs []span
		at := 0
		for _, one := range atoms {
			n := len([]rune(one.text))
			if one.isUnbroken {
				runs = append(runs, span{at, at + n})
			}
			at += n
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
			cut++
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
	// The property says where a cut may fall, so a run of it that never cut
	// anything, or never met a link straddling the place the cut falls, has
	// asked nothing.
	if cut < 25 || straddled < 15 {
		t.Fatalf("%d lines were cut and %d of those had a run standing across the cut",
			cut, straddled)
	}
}
