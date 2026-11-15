// Package textplace reads the place of a search result in the numbering of its source, and writes the
// address of a result.
package textplace

import (
	"encoding/json"
	"fmt"
	"net/url"
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

// GetSelector reads the JSON selector of a passage; an empty text or a selector that cannot be read is an
// empty selector.
func GetSelector(raw string) Selector {
	var s Selector
	if raw == "" {
		return s
	}
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Selector{Page: 0, CFI: "", T: nil}
	}
	return s
}

// FormatAddress writes the address of the passage of the source, which a markdown link targets.
func FormatAddress(source string, start, length int) string {
	return fmt.Sprintf("numen:%s?start=%d&length=%d", url.PathEscape(source), start, length)
}

// FormatNoteAddress writes the address of a whole note, which a markdown link targets.
func FormatNoteAddress(notePath string) string {
	return "numen:" + url.PathEscape(notePath)
}
