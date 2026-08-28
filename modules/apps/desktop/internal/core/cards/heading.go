package cards

import (
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/mark"
)

// HeadingRunes is how many characters a card's heading is cut to. It is counted
// in characters and not in bytes, so a heading of one script is as long as a
// heading of another.
const HeadingRunes = 120

// caret is what a mark is written behind, at the end of a card's heading.
const caret = "^"

// ReadHeading reads a card's heading — the text after the hashes — into what it
// shows and the mark it carries.
//
// A mark is separated from the text by one space, stands last, and is a mark
// only at the length and in the alphabet marks are minted in. Anything else at
// the end of a heading is heading text, and comes back as part of the text.
func ReadHeading(heading string) (text, carried string) {
	at := strings.LastIndex(heading, " ")
	last := heading
	if at >= 0 {
		last = heading[at+1:]
	}
	written, found := strings.CutPrefix(last, caret)
	if !found || !mark.Valid(written) {
		return heading, ""
	}
	if at < 0 {
		return "", written
	}
	return heading[:at], written
}

// WriteHeading is the heading a card of this text and this mark stands under.
func WriteHeading(text, carried string) string {
	switch {
	case carried == "":
		return text
	case text == "":
		return caret + carried
	}
	return text + " " + caret + carried
}

// Project is the heading text a card's first field gives. The heading holds
// nothing of its own: throw it away and this writes it again from the field.
//
// It is taken over text whose line endings are normalised. It stops at the
// first line break and at HeadingRunes characters, and the spaces at either end
// are dropped. A field that is empty, or holds only spaces, projects to nothing.
func Project(value string) string {
	line := markdown.Normalised(value)
	if at := strings.IndexByte(line, '\n'); at >= 0 {
		line = line[:at]
	}
	line = strings.TrimSpace(line)

	cut, runes := len(line), 0
	for at := range line {
		if runes == HeadingRunes {
			cut = at
			break
		}
		runes++
	}
	if cut == len(line) {
		return line
	}
	// The cut never falls inside one of these: it falls in front of the whole
	// of whichever it lands in.
	for _, span := range unbroken(line) {
		if span[0] < cut && cut < span[1] {
			cut = span[0]
		}
	}
	return strings.TrimRight(line[:cut], " \t")
}

// linkRe is a wikilink or an embed of one, and emphasisRe is a run of emphasis
// closed by the delimiter it was opened with.
var (
	linkRe     = regexp.MustCompile(`!?\[\[[^\[\]]*\]\]`)
	emphasisRe = regexp.MustCompile(`\*\*\*[^*]+\*\*\*|\*\*[^*]+\*\*|\*[^*]+\*|___[^_]+___|__[^_]+__|_[^_]+_`)
)

// unbroken is every run of the line a cut may not fall inside, in the order
// they stand. A link's brackets hold whatever a person wrote, so what is inside
// one is the link's and is looked at no further.
func unbroken(line string) [][]int {
	var out [][]int
	at := 0
	for _, span := range linkRe.FindAllStringIndex(line, -1) {
		out = append(out, emphasis(line[at:span[0]], at)...)
		out = append(out, span)
		at = span[1]
	}
	return append(out, emphasis(line[at:], at)...)
}

// emphasis is every run of emphasis in one stretch of the line, counted from
// the byte that stretch begins at.
func emphasis(stretch string, from int) [][]int {
	var out [][]int
	for _, span := range emphasisRe.FindAllStringIndex(stretch, -1) {
		out = append(out, []int{from + span[0], from + span[1]})
	}
	return out
}
