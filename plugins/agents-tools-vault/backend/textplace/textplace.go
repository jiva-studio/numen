// Package textplace locates the bytes of a text version in the numbering of its source: the pages of a
// PDF, the positions of a book and the seconds of a recording.
package textplace

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Selector has the keys of a segment's selector that identify a location: a page of a PDF counted from 1,
// the position in a book, and the span of a recording in seconds.
type Selector struct {
	Page int       `json:"page"`
	CFI  string    `json:"cfi"`
	T    []float64 `json:"t"`
}

// IsEmpty reports whether the selector names no page, position or time.
func (s Selector) IsEmpty() bool {
	return s.Page == 0 && s.CFI == "" && len(s.T) == 0
}

// Span is a segment of a text: its byte range and its location in the numbering of the source.
type Span struct {
	Start, End int
	Selector   Selector
}

// Range is a range of bytes of a text and the location where it starts.
type Range struct {
	Start, Length int
	Selector      Selector
}

// ParseSelector reads the JSON selector of a segment; an empty text is an empty selector.
func ParseSelector(raw string) (Selector, error) {
	var s Selector
	if raw == "" {
		return s, nil
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Selector{Page: 0, CFI: "", T: nil}, err
	}
	return s, nil
}

// GetSelector reads the JSON selector of a passage; a selector that cannot be read is an empty one.
func GetSelector(raw string) Selector {
	s, err := ParseSelector(raw)
	if err != nil {
		return Selector{Page: 0, CFI: "", T: nil}
	}
	return s
}

// GetSpans locates the segments, which start at the byte offset, and fails on a selector that cannot be read.
func GetSpans(segments []domain.TextSegment, offset int) ([]Span, error) {
	spans := make([]Span, 0, len(segments))
	for _, segment := range segments {
		s, err := ParseSelector(string(segment.Selector))
		if err != nil {
			return nil, fmt.Errorf("segment at byte %d has an invalid selector %q: %w", offset, segment.Selector, err)
		}
		spans = append(spans, Span{Start: offset, End: offset + len(segment.Text), Selector: s})
		offset += len(segment.Text)
	}
	return spans, nil
}

// IsPastPage reports whether the spans hold the page and a page after it.
func IsPastPage(spans []Span, page int) bool {
	isSeen := false
	for _, span := range spans {
		isSeen = isSeen || span.Selector.Page == page
		if isSeen && span.Selector.Page > page {
			return true
		}
	}
	return false
}

// FindPage returns the byte range of the page, or says which pages the spans have.
func FindPage(page int, spans []Span) (start, end int, err error) {
	first, last := 0, 0
	start, end = -1, -1
	for _, span := range spans {
		if span.Selector.Page == 0 {
			continue
		}
		if first == 0 || span.Selector.Page < first {
			first = span.Selector.Page
		}
		last = max(last, span.Selector.Page)
		if span.Selector.Page == page {
			if start < 0 {
				start = span.Start
			}
			end = span.End
		}
	}
	switch {
	case first == 0:
		return 0, 0, errors.New("the text has no pages; read it by start and length")
	case start < 0:
		return 0, 0, fmt.Errorf("the text has no page %d: its pages are %d to %d", page, first, last)
	}
	return start, end, nil
}

// GetRanges cuts the spans to the bytes from rangeStart to rangeEnd, dropping the spans with no location
// and joining the neighbors on one page.
func GetRanges(spans []Span, rangeStart, rangeEnd int) []Range {
	var out []Range
	for _, span := range spans {
		start, end := max(span.Start, rangeStart), min(span.End, rangeEnd)
		if start >= end || span.Selector.IsEmpty() {
			continue
		}
		if last := len(out) - 1; last >= 0 && span.Selector.Page > 0 && out[last].Selector.Page == span.Selector.Page {
			out[last].Length = end - out[last].Start
			continue
		}
		out = append(out, Range{Start: start, Length: end - start, Selector: span.Selector})
	}
	return out
}

// GetNextStart returns the byte where a read continues, or nil when end is the end of a text of size bytes.
func GetNextStart(end, size int) *int {
	if end >= size {
		return nil
	}
	return &end
}

// CheckStart rejects a start past the end of a text of size bytes.
func CheckStart(start, size int) error {
	if start > size || (start == size && size > 0) {
		return fmt.Errorf("start %d is past the end of the text, which is %d bytes long", start, size)
	}
	return nil
}

// CheckRead rejects a negative page, start or length, and a page together with a start.
func CheckRead(page, start, length int) error {
	for _, field := range []struct {
		name  string
		value int
	}{{"page", page}, {"start", start}, {"length", length}} {
		if field.value < 0 {
			return fmt.Errorf("%s is %d: give a number of zero or more", field.name, field.value)
		}
	}
	if page > 0 && start > 0 {
		return errors.New("give a page, or a start and a length, not a page and a start")
	}
	return nil
}

// FormatAddress writes the address of the passage of the source, which a markdown link targets.
func FormatAddress(source string, start, length int) string {
	return fmt.Sprintf("numen:%s?start=%d&length=%d", url.PathEscape(source), start, length)
}
