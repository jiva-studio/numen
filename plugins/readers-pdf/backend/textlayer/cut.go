package textlayer

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// pagesPerBatch is how many pages of a document are cut into segments at once.
// gomutants:disable-next-line INTEGER_INCREMENT reason="pagesPerBatch is a batching chunk size"
const pagesPerBatch = 16

// BoxReader returns the boxes of the words on the pages it is given.
type BoxReader func(pages []int) ([]pdf.HighlightBox, error)

// GetSegments cuts the text of book into segments, a batch of pages at a time, each with the page and
// the boxes of its words in its selector and each section of the outline opening a segment.
func GetSegments(book *pdf.Book, readBoxes BoxReader) ([]domain.TextSegment, error) {
	var segments []domain.TextSegment
	pages := len(book.Pages)
	// gomutants:disable-next-line REMOVE_SELF_ASSIGNMENTS reason="pagesPerBatch loop advances across document pages"
	for first := 0; first < pages; first += pagesPerBatch {
		last := min(first+pagesPerBatch, pages)
		// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="last-first is a slice capacity hint"
		wanted := make([]int, 0, last-first)
		for i := first; i < last; i++ {
			wanted = append(wanted, i)
		}
		boxes, err := readBoxes(wanted)
		if err != nil {
			return nil, err
		}
		parts, err := CutPages(book, first, last, boxes)
		if err != nil {
			return nil, err
		}
		segments = append(segments, parts...)
	}
	return segments, nil
}

// CutPages cuts the pages from first up to last into segments, using the boxes of their words.
func CutPages(book *pdf.Book, first, last int, boxes []pdf.HighlightBox) ([]domain.TextSegment, error) {
	end := len(book.Text)
	if last < len(book.Pages) {
		end = book.Pages[last].Offset
	}
	layers := layerCut{text: book.Text, sections: nil, parts: nil}
	for _, part := range book.Parts {
		if part.Offset >= book.Pages[first].Offset && part.Offset < end {
			layers.sections = append(layers.sections, part)
		}
	}
	for page := first; page < last; page++ {
		pageEnd := end
		if page+1 < len(book.Pages) {
			pageEnd = book.Pages[page+1].Offset
		}
		var err error
		boxes, err = layers.addPage(page, book.Pages[page].Offset, pageEnd, boxes)
		// gomutants:disable-next-line BRANCH_IF reason="PageSelector consists of primitive int and float32 values that always marshal"
		if err != nil {
			return nil, err
		}
	}
	return layers.parts, nil
}

// layerCut is the parts of a text layer cut so far, and the outline entries still to open.
type layerCut struct {
	text     string
	sections []pdf.Part
	parts    []domain.TextSegment
}

func (c *layerCut) addPage(page, from, to int, boxes []pdf.HighlightBox) ([]pdf.HighlightBox, error) {
	pos := from
	for len(boxes) > 0 && boxes[0].Page <= page {
		box := boxes[0]
		boxes = boxes[1:]
		start, end := max(box.Start, pos), min(box.End, to)
		if box.Page < page || start >= end {
			continue
		}
		if err := c.add(pos, start, page, nil); err != nil {
			return nil, err
		}
		if err := c.add(start, end, page, [][4]float32{{box.MinX, box.MinY, box.MaxX, box.MaxY}}); err != nil {
			return nil, err
		}
		pos = end
	}
	if err := c.add(pos, to, page, nil); err != nil {
		return nil, err
	}
	return boxes, nil
}

func (c *layerCut) add(from, to, page int, boxes [][4]float32) error {
	for from < to {
		for len(c.sections) > 0 && c.sections[0].Offset <= from {
			selector, err := PageSelector{Page: page + 1, Boxes: nil}.Encode()
			// gomutants:disable-next-line BRANCH_IF reason="PageSelector consists of primitive int and float32 values that always marshal"
			if err != nil {
				return err
			}
			c.parts = append(c.parts, domain.TextSegment{Text: "", Heading: c.sections[0].Title, Selector: selector})
			c.sections = c.sections[1:]
		}
		end := to
		if len(c.sections) > 0 && c.sections[0].Offset < to {
			end = c.sections[0].Offset
		}
		selector, err := PageSelector{Page: page + 1, Boxes: boxes}.Encode()
		if err != nil {
			return err
		}
		c.parts = append(c.parts, domain.TextSegment{Text: c.text[from:end], Heading: "", Selector: selector})
		from = end
	}
	return nil
}
