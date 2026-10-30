package link

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// timeSelector locates a segment in a recording: T is [start, end] in seconds from the start of
// the recording (W3C Media Fragments).
type timeSelector struct {
	T []float64 `json:"t,omitempty"`
}

// encode returns the selector as the JSON a segment carries.
func (s timeSelector) encode() domain.Selector {
	// A struct of numbers and slices of them always marshals.
	raw, _ := json.Marshal(s)
	return domain.Selector(raw)
}

// readTimeSelector reads the span of a segment's selector. An empty selector has none.
func readTimeSelector(raw domain.Selector) (timeSelector, error) {
	var out timeSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
