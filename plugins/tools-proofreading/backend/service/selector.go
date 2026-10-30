package service

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// segmentSelector is the keys of a segment's selector that proofreading carries from the text it
// reads to the text it writes (W3C Web Annotation selector): T is [start, end] in seconds from the
// start of a recording, Page counts from 1, CFI is an EPUB Canonical Fragment Identifier, and Boxes
// are rectangles on the page in fractions of it.
type segmentSelector struct {
	T     []float64    `json:"t,omitempty"`
	Page  int          `json:"page,omitempty"`
	CFI   string       `json:"cfi,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

// encode returns the selector as the JSON a segment carries.
func (s segmentSelector) encode() domain.Selector {
	// A struct of numbers, strings and slices of them always marshals.
	raw, _ := json.Marshal(s)
	return domain.Selector(raw)
}

// readSegmentSelector reads the keys of a segment's selector. An empty selector has none.
func readSegmentSelector(raw domain.Selector) (segmentSelector, error) {
	var out segmentSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
