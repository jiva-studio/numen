package domain

import (
	"encoding/json"
	"fmt"
)

// PageSelector locates a segment on a page of a document (W3C Web Annotation selector). Page counts
// from 1 (RFC 8118); Boxes are rectangles on the page, each [minX, minY, maxX, maxY] in fractions of
// the page.
type PageSelector struct {
	Page  int          `json:"page,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

// TimeSelector locates a segment in a recording: T is [start, end] in seconds from the start of the
// recording (W3C Media Fragments).
type TimeSelector struct {
	T []float64 `json:"t,omitempty"`
}

// CFISelector locates a segment in a book: CFI is an EPUB Canonical Fragment Identifier.
type CFISelector struct {
	CFI string `json:"cfi,omitempty"`
}

// EncodeSelector returns the value as the JSON a segment carries.
func EncodeSelector[T PageSelector | TimeSelector | CFISelector](value T) (Selector, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %T: %w", value, err)
	}
	return Selector(raw), nil
}

// DecodeSelector reads the keys of a segment's selector. An empty selector has none.
func DecodeSelector[T PageSelector | TimeSelector | CFISelector](raw Selector) (T, error) {
	var out T
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
