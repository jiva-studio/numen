// Package selector writes the selectors that locate a segment of a text version.
package selector

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Time locates a segment in a recording: T is [start, end] in seconds from the start of the
// recording (W3C Media Fragments).
type Time struct {
	T []float64 `json:"t,omitempty"`
}

// Encode returns the selector as the host keeps it.
func (s Time) Encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return domain.Selector(raw), nil
}
