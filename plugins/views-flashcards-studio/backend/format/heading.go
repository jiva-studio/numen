package format

import (
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardid"
)

// HeadingRunes is how many characters a card's heading is cut to. It is counted
// in characters and not in bytes, so a heading of one script is as long as a
// heading of another.
const HeadingRunes = 120

// caret is what a card id is written behind, at the end of a card's heading.
const caret = "^"

// ReadHeading reads a card's heading — the text after the hashes — into what it
// shows and the card id it carries.
//
// A card id is separated from the text by one space, stands last, and is a card id
// only at the length and in the alphabet card ids are written in. Anything else at
// the end of a heading is heading text, and comes back as part of the text.
func ReadHeading(heading string) (text string, carried cardid.ID) {
	pos := strings.LastIndex(heading, " ")
	last := heading
	if pos >= 0 {
		last = heading[pos+1:]
	}
	content, found := strings.CutPrefix(last, caret)
	if !found || !cardid.IsValid(cardid.ID(content)) {
		return heading, ""
	}
	if pos < 0 {
		return "", cardid.ID(content)
	}
	return heading[:pos], cardid.ID(content)
}

// WriteHeading is the heading a card of this text and this card id is placed under.
func WriteHeading(text string, carried cardid.ID) string {
	switch {
	case carried == "":
		return text
	case text == "":
		return caret + string(carried)
	}
	return text + " " + caret + string(carried)
}

// Project is the heading text a card's first field gives. The heading holds
// nothing of its own: throw it away and this writes it again from the field.
//
// It is taken over text whose line endings are normalised. It stops at the
// first line break and at HeadingRunes characters, and the spaces at either end
// are dropped. A field that is empty, or holds only spaces, projects to nothing.
func Project(value string) string {
	line := markdown.Normalise(value)
	if pos := strings.IndexByte(line, '\n'); pos >= 0 {
		line = line[:pos]
	}
	line = strings.TrimSpace(line)

	trimmed, runes := len(line), 0
	for item := range line {
		if runes == HeadingRunes {
			trimmed = item
			break
		}
		runes++
	}
	if trimmed == len(line) {
		return line
	}
	// The cut never falls inside one of these: it falls in front of the whole
	// moved until it stops moving.
	spans := getUnsplittableRanges(line)
	for moved := true; moved; {
		moved = false
		for _, span := range spans {
			if span[0] < trimmed && trimmed < span[1] {
				trimmed, moved = span[0], true
			}
		}
	}
	return strings.TrimRight(line[:trimmed], " \t")
}

// emphasisRe is a run of emphasis closed by the delimiter it was opened with.
var emphasisRe = regexp.MustCompile(
	`\*\*\*[^*]+\*\*\*|\*\*[^*]+\*\*|\*[^*]+\*|___[^_]+___|__[^_]+__|_[^_]+_`)

// maskRune is what a link's bytes are read as while a run of emphasis is looked
// for: one character that opens and closes nothing.
const maskRune = 'x'

// getUnsplittableRanges is every run of the line a cut may not fall inside: its links, and
// its runs of emphasis. A link's brackets hold whatever a person wrote, so the
// runs of emphasis are looked for over the line with every link covered over,
// and a run holding a link is one run.
func getUnsplittableRanges(line string) [][]int {
	found := links(line)
	masked := []byte(line)
	for _, span := range found {
		for pos := span[0]; pos < span[1]; pos++ {
			masked[pos] = maskRune
		}
	}
	return append(found, emphasisRe.FindAllIndex(masked, -1)...)
}

func matchLinkEnd(line string, pos int) (int, bool) {
	depth, to := 0, pos
	for to+1 < len(line) {
		switch {
		case line[to] == '[' && line[to+1] == '[':
			depth, to = depth+1, to+2
		case line[to] == ']' && line[to+1] == ']':
			depth, to = depth-1, to+2
		default:
			to++
			continue
		}
		if depth == 0 {
			return to, true
		}
	}
	return to, false
}

func linkStart(line string, pos int) int {
	if pos > 0 && line[pos-1] == '!' {
		return pos - 1
	}
	return pos
}

// links is every wikilink and embed of the line. The brackets are counted, so a
// link holding another closes on its own last pair, and a pair that is never
// closed is no link and neither is anything after it.
func links(line string) [][]int {
	var out [][]int
	for pos := 0; pos+1 < len(line); {
		if line[pos] != '[' || line[pos+1] != '[' {
			pos++
			continue
		}
		head := linkStart(line, pos)
		to, ok := matchLinkEnd(line, pos)
		if !ok {
			return out
		}
		out = append(out, []int{head, to})
		pos = to
	}
	return out
}

// oneLine is what a name a caller composed a heading from stands as. A heading
func oneLine(name string) string {
	line := markdown.Normalise(name)
	if pos := strings.IndexByte(line, '\n'); pos >= 0 {
		line = line[:pos]
	}
	return strings.TrimSpace(line)
}

// headingLine is the line a card, a template, a field or a side is placed under. A
// heading carrying no text is the hashes and nothing after them.
func headingLine(level int, name string) string {
	hashes := strings.Repeat("#", level)
	if name == "" {
		return hashes
	}
	return hashes + " " + name
}
