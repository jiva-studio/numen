package recognition

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// pageSelector locates a segment on a page of the recognised file (W3C Web Annotation selector).
// Page counts from 1 (RFC 8118); Boxes are rectangles on the page, each [minX, minY, maxX, maxY]
// in fractions of the page.
type pageSelector struct {
	Page  int          `json:"page,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

// encode returns the selector as the JSON a segment carries.
func (s pageSelector) encode() domain.Selector {
	// A struct of numbers and slices of them always marshals.
	raw, _ := json.Marshal(s)
	return domain.Selector(raw)
}

// readPageSelector reads the page keys of a segment's selector. An empty selector has none.
func readPageSelector(raw domain.Selector) (pageSelector, error) {
	var out pageSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
