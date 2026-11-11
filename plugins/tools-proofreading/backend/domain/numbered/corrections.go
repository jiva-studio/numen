package numbered

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// ApplyCorrections puts the corrected lines in the place of the segments they cover. A line that
// joins a run makes one segment that takes the selector of the whole run.
func ApplyCorrections(parts []domain.TextSegment, corrections []Line) []domain.TextSegment {
	byLine := make(map[int]Line, len(corrections))
	for _, line := range corrections {
		byLine[line.Number] = line
	}
	out := make([]domain.TextSegment, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		line, ok := byLine[i+1]
		if !ok {
			out = append(out, parts[i])
			continue
		}
		last := min(line.GetLastNumber(), len(parts)) - 1
		first, end := parts[i], parts[last]
		part := domain.TextSegment{
			Text:     getLeadingSpace(first.Text) + line.Text + getTrailingSpace(end.Text),
			Heading:  first.Heading,
			Selector: first.Selector,
		}
		if last > i {
			part.Selector = joinSelectors(parts[i : last+1])
		}
		out = append(out, part)
		i = last
	}
	return out
}

func getLeadingSpace(text string) string {
	return text[:len(text)-len(strings.TrimLeft(text, " \t\r\n"))]
}

func getTrailingSpace(text string) string {
	return text[len(strings.TrimRight(text, " \t\r\n")):]
}

func joinSelectors(run []domain.TextSegment) domain.Selector {
	first, err := ReadSelector(run[0].Selector)
	if err != nil {
		return run[0].Selector
	}
	joined := Selector{T: nil, Page: first.Page, CFI: first.CFI, Boxes: nil}
	for _, one := range run {
		selector, err := ReadSelector(one.Selector)
		if err != nil {
			continue
		}
		if len(selector.T) == 2 {
			if len(joined.T) == 0 {
				joined.T = []float64{selector.T[0], selector.T[1]}
			}
			joined.T[1] = selector.T[1]
		}
		if selector.Page == first.Page {
			joined.Boxes = append(joined.Boxes, selector.Boxes...)
		}
	}
	encoded, err := joined.Encode()
	if err != nil {
		return run[0].Selector
	}
	return encoded
}
