package format

import (
	"regexp"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
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
// only at the length and in the alphabet marks are written in. Anything else at
// the end of a heading is heading text, and comes back as part of the text.
func ReadHeading(heading string) (text string, carried domain.CardID) {
	at := strings.LastIndex(heading, " ")
	last := heading
	if at >= 0 {
		last = heading[at+1:]
	}
	written, found := strings.CutPrefix(last, caret)
	if !found || !cardid.Valid(domain.CardID(written)) {
		return heading, ""
	}
	if at < 0 {
		return "", domain.CardID(written)
	}
	return heading[:at], domain.CardID(written)
}

// WriteHeading is the heading a card of this text and this mark stands under.
func WriteHeading(text string, carried domain.CardID) string {
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
	// of whichever it lands in. One of them stands inside another, so the cut is
	// moved until it stops moving.
	spans := unbroken(line)
	for moved := true; moved; {
		moved = false
		for _, span := range spans {
			if span[0] < cut && cut < span[1] {
				cut, moved = span[0], true
			}
		}
	}
	return strings.TrimRight(line[:cut], " \t")
}

// emphasisRe is a run of emphasis closed by the delimiter it was opened with.
var emphasisRe = regexp.MustCompile(
	`\*\*\*[^*]+\*\*\*|\*\*[^*]+\*\*|\*[^*]+\*|___[^_]+___|__[^_]+__|_[^_]+_`)

// covered is what a link's bytes are read as while a run of emphasis is looked
// for: one character that opens and closes nothing.
const covered = 'x'

// unbroken is every run of the line a cut may not fall inside: its links, and
// its runs of emphasis. A link's brackets hold whatever a person wrote, so the
// runs of emphasis are looked for over the line with every link covered over,
// and a run holding a link is one run.
func unbroken(line string) [][]int {
	found := links(line)
	masked := []byte(line)
	for _, span := range found {
		for at := span[0]; at < span[1]; at++ {
			masked[at] = covered
		}
	}
	return append(found, emphasisRe.FindAllIndex(masked, -1)...)
}

// links is every wikilink and embed of the line. The brackets are counted, so a
// link holding another closes on its own last pair, and a pair that is never
// closed is no link and neither is anything after it.
func links(line string) [][]int {
	var out [][]int
	for at := 0; at+1 < len(line); {
		if line[at] != '[' || line[at+1] != '[' {
			at++
			continue
		}
		head := at
		if head > 0 && line[head-1] == '!' {
			head--
		}
		depth, to := 0, at
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
				break
			}
		}
		if depth != 0 {
			return out
		}
		out = append(out, []int{head, to})
		at = to
	}
	return out
}

// oneLine is what a name a caller composed a heading from stands as. A heading
// is one line, so it holds what stands in front of the first break in it.
func oneLine(name string) string {
	line := markdown.Normalise(name)
	if at := strings.IndexByte(line, '\n'); at >= 0 {
		line = line[:at]
	}
	return strings.TrimSpace(line)
}

// headingLine is the line a card, a face, a field or a side stands under. A
// heading carrying no text is the hashes and nothing after them.
func headingLine(level int, name string) string {
	hashes := strings.Repeat("#", level)
	if name == "" {
		return hashes
	}
	return hashes + " " + name
}
