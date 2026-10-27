// Package textfile reads and writes the file of a text version: JSON Lines,
// one segment per line, {"text", "heading"?, "selector"?}. Every plugin writes
// this format into its folder, and the core reads it.
package textfile

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidSelector is a selector that is not JSON.
var ErrInvalidSelector = errors.New("textfile: the selector is not JSON")

// line is one segment as a line of the file holds it.
type line struct {
	Text     string          `json:"text"`
	Heading  string          `json:"heading,omitempty"`
	Selector json.RawMessage `json:"selector,omitempty"`
}

// Encode is the segments as the lines of a version file, each ending in a
// newline. Appending the encoding of more segments to a file appends them.
func Encode(segments []domain.TextSegment) ([]byte, error) {
	var b bytes.Buffer
	for i, s := range segments {
		one := line{Text: s.Text, Heading: s.Heading}
		if s.Selector != "" {
			if !json.Valid([]byte(s.Selector)) {
				return nil, fmt.Errorf("segment %d: %w", i, ErrInvalidSelector)
			}
			one.Selector = json.RawMessage(s.Selector)
		}
		raw, err := json.Marshal(one)
		if err != nil {
			return nil, fmt.Errorf("segment %d: %w", i, err)
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
}

// Decode is the segments of a version file, in order. Blank lines hold no segment.
func Decode(raw []byte) ([]domain.TextSegment, error) {
	var segments []domain.TextSegment
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(nil, len(raw)+1)
	for number := 1; scanner.Scan(); number++ {
		text := bytes.TrimSpace(scanner.Bytes())
		if len(text) == 0 {
			continue
		}
		var one line
		if err := json.Unmarshal(text, &one); err != nil {
			return nil, fmt.Errorf("textfile: line %d: %w", number, err)
		}
		segment := domain.TextSegment{Text: one.Text, Heading: one.Heading}
		if len(one.Selector) > 0 && !bytes.Equal(one.Selector, []byte("null")) {
			segment.Selector = domain.Selector(one.Selector)
		}
		segments = append(segments, segment)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("textfile: %w", err)
	}
	return segments, nil
}
