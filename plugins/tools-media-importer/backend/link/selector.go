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
func (s timeSelector) encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return domain.Selector(raw), nil
}
