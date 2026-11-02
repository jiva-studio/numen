package transcript

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// timeSelector locates a segment in a recording: T is [start, end] in seconds from the start of
// the recording (W3C Media Fragments).
type timeSelector struct {
	T []float64 `json:"t,omitempty"`
}

// encode returns the selector as the JSON a segment carries.
func (s timeSelector) encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode time selector: %w", err)
	}
	return domain.Selector(raw), nil
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
