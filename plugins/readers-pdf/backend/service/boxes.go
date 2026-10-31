package service

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// getBoxes returns the rectangles of the segments a run of a version's text overlaps, in the order
// of the segments. Each box carries the page its selector names, counted from zero, and the run of
// the segment it covers. A segment whose selector names no page has no boxes.
func getBoxes(text host.VersionText, run pdf.ByteRange) []pdf.HighlightBox {
	if run.Start >= run.End {
		return nil
	}
	var boxes []pdf.HighlightBox
	from := 0
	for _, segment := range text.GetSegments() {
		to := from + len(segment.Text)
		if to > run.Start && from < run.End {
			if selector, err := domain.DecodeSelector[domain.PageSelector](segment.Selector); err == nil && selector.Page > 0 {
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
