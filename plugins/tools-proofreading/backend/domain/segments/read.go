package segments

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

// ReadLines numbers the segments from 1 as lines, without the white space around their words.
func ReadLines(parts []domain.TextSegment) []numbered.Line {
	lines := make([]numbered.Line, len(parts))
	for i, part := range parts {
		lines[i] = numbered.Line{Number: i + 1, Last: 0, Text: strings.TrimSpace(part.Text)}
	}
	return lines
}

// ReadPages groups the lines into one batch per page of their selectors.
func ReadPages(parts []domain.TextSegment, lines []numbered.Line) []numbered.Batch {
	var out []numbered.Batch
	for _, line := range lines {
		selector, err := ReadSelector(parts[line.Number-1].Selector)
		if err != nil {
			selector = Selector{T: nil, Page: 0, CFI: "", Boxes: nil}
		}
		page := selector.Page
		if len(out) == 0 || out[len(out)-1].Number != page {
			out = append(out, numbered.Batch{Number: page, Lines: nil, IsJoinable: false, Context: ""})
		}
		out[len(out)-1].Lines = append(out[len(out)-1].Lines, line)
	}
	return out
}

// IsTimed reports whether the first segment is placed by seconds in a recording.
func IsTimed(parts []domain.TextSegment) bool {
	if len(parts) == 0 {
		return false
	}
	selector, err := ReadSelector(parts[0].Selector)
	return err == nil && len(selector.T) == 2
}
