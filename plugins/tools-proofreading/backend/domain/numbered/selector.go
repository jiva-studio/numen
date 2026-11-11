package numbered

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Selector is the keys of a segment's selector that proofreading carries from the text it
// reads to the text it writes (W3C Web Annotation selector): T is [start, end] in seconds from the
// start of a recording, Page counts from 1, CFI is an EPUB Canonical Fragment Identifier, and Boxes
// are rectangles on the page in fractions of it.
type Selector struct {
	T     []float64    `json:"t,omitempty"`
	Page  int          `json:"page,omitempty"`
	CFI   string       `json:"cfi,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

func (s Selector) Encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return domain.Selector(raw), nil
}

func ReadSelector(raw domain.Selector) (Selector, error) {
	var out Selector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
