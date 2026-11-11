// Package segments cuts the text of a book into the segments of a text version: each segment is at most
// PartBytes long, begins on a whole character, and is selected by the CFI of its first character. The
// sections the book names open at the segment where they begin.
package segments

import (
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub"
)

// PartBytes is the most text one segment holds.
const PartBytes = 64 << 10

// Cut returns the segments of text. Each section heading is a segment of its own, placed before the
// text segment that starts at the section's offset.
func Cut(text string, sections []epub.Part, places epub.Places) ([]domain.TextSegment, error) {
	cutter := segmentCut{text: text, sections: sections, blocks: places.Blocks}
	var segments []domain.TextSegment
	from := 0
	for {
		selector, err := getSelector(places, from)
		// gomutants:disable-next-line BRANCH_IF reason="cfiSelector consists of a string field that always marshals"
		if err != nil {
			return nil, err
		}
		for _, title := range cutter.openAt(from) {
			segments = append(segments, domain.TextSegment{Text: "", Heading: title, Selector: selector})
		}
		if from >= len(text) {
			return segments, nil
		}
		to := cutter.getEnd(from)
		segments = append(segments, domain.TextSegment{Text: text[from:to], Heading: "", Selector: selector})
		from = to
	}
}

func getSelector(places epub.Places, offset int) (domain.Selector, error) {
	place, ok := places.GetCFI(offset)
	if !ok {
		return "", nil
	}
	return cfiSelector{CFI: place.String()}.encode()
}

// segmentCut is the text of a book, and the sections and block starts still ahead of the cut.
type segmentCut struct {
	text     string
	sections []epub.Part
	blocks   []int
}

func (c *segmentCut) openAt(from int) []string {
	for len(c.blocks) > 0 && c.blocks[0] <= from {
		c.blocks = c.blocks[1:]
	}
	var titles []string
	for len(c.sections) > 0 && c.sections[0].Offset <= from {
		titles = append(titles, c.sections[0].Title)
		c.sections = c.sections[1:]
	}
	return titles
}

func (c *segmentCut) getEnd(from int) int {
	to := getCutEnd(c.text, from)
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="setting to = c.blocks[0] when equal is a no-op"
	if len(c.blocks) > 0 && c.blocks[0] < to {
		to = c.blocks[0]
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="setting to = c.sections[0].Offset when equal is a no-op"
	if len(c.sections) > 0 && c.sections[0].Offset < to {
		to = c.sections[0].Offset
	}
	return to
}

func getCutEnd(text string, from int) int {
	end := min(from+PartBytes, len(text))
	to := end
	// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION,INVERT_LOGICAL reason="text[from] is always a rune start and PartBytes exceeds any UTF-8 rune length"
	for to > from && to < len(text) && !utf8.RuneStart(text[to]) {
		to--
	}
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="PartBytes is 65536 so backing up at most 3 bytes never reaches from"
	if to == from {
		return end
	}
	return to
}
