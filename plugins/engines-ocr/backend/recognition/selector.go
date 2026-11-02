package recognition

import (
	"encoding/json"
	"fmt"

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
func (s pageSelector) encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode page selector: %w", err)
	}
	return domain.Selector(raw), nil
}
