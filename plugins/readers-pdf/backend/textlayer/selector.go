// Package textlayer holds the rules that tie the text of a PDF file to its pages: the selector of a
// segment, the segments cut from the pages and the boxes a run of text covers.
package textlayer

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// PageSelector locates a segment on a page of the document (W3C Web Annotation selector). Page
// counts from 1 (RFC 8118); Boxes are rectangles on the page, each [minX, minY, maxX, maxY] in
// fractions of the page.
type PageSelector struct {
	Page  int          `json:"page,omitempty"`
	Boxes [][4]float32 `json:"boxes,omitempty"`
}

// Encode returns the selector as the host keeps it.
func (s PageSelector) Encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode page selector: %w", err)
	}
	return domain.Selector(raw), nil
}

// ReadPageSelector returns the selector raw holds; an empty selector has no page.
func ReadPageSelector(raw domain.Selector) (PageSelector, error) {
	var out PageSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}

// GetBoxes returns the boxes of the segments of text that run covers.
func GetBoxes(text domain.VersionText, run pdf.ByteRange) []pdf.HighlightBox {
	if run.Start >= run.End {
		return nil
	}
	var boxes []pdf.HighlightBox
	from := 0
	for _, segment := range text.GetSegments() {
		to := from + len(segment.Text)
		if to > run.Start && from < run.End {
			if selector, err := ReadPageSelector(segment.Selector); err == nil && selector.Page > 0 {
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
