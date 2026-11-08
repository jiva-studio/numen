package service

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// pageSelector locates a segment on a page of the document (W3C Web Annotation selector). Page
// counts from 1 (RFC 8118); Boxes are rectangles on the page, each [minX, minY, maxX, maxY] in
// fractions of the page.
type pageSelector struct {
	Page  int          `json:"page,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

func (s pageSelector) encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode page selector: %w", err)
	}
	return domain.Selector(raw), nil
}

func readPageSelector(raw domain.Selector) (pageSelector, error) {
	var out pageSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}

func getBoxes(text domain.VersionText, run pdf.ByteRange) []pdf.HighlightBox {
	if run.Start >= run.End {
		return nil
	}
	var boxes []pdf.HighlightBox
	from := 0
	for _, segment := range text.GetSegments() {
		to := from + len(segment.Text)
		if to > run.Start && from < run.End {
			if selector, err := readPageSelector(segment.Selector); err == nil && selector.Page > 0 {
				for _, box := range selector.Boxes {
					boxes = append(boxes, pdf.HighlightBox{
						Page:      selector.Page - 1,
						ByteRange: pdf.ByteRange{Start: from, End: to},
						Rect:      pdf.Rect{MinX: box[0], MinY: box[1], MaxX: box[2], MaxY: box[3]},
					})
				}
			}
		}
		from = to
	}
	return boxes
}
