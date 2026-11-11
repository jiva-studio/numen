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

	"github.com/jiva-studio/numen/libs/note/domain"
)

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
		// gomutants:disable-next-line BRANCH_IF reason="json.Marshal on strings and a validated RawMessage cannot fail"
		if err != nil {
			// gomutants:disable-next-line RETURN_ERROR_NIL,ERRORF_WRAP reason="json.Marshal on strings and a validated RawMessage cannot fail"
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
	// gomutants:disable-next-line INTEGER_INCREMENT reason="a larger scanner buffer ceiling does not alter token scanning"
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
		// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="converting an empty slice to domain.Selector yields the same empty string"
		if len(one.Selector) > 0 && !bytes.Equal(one.Selector, []byte("null")) {
			segment.Selector = domain.Selector(one.Selector)
		}
		segments = append(segments, segment)
	}
	// gomutants:disable-next-line BRANCH_IF reason="bytes.Reader with a buffer of len(raw)+1 never produces a scanner error"
	if err := scanner.Err(); err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL,ERRORF_WRAP reason="bytes.Reader with a buffer of len(raw)+1 never produces a scanner error"
		return nil, fmt.Errorf("textfile: %w", err)
	}
	return segments, nil
}
