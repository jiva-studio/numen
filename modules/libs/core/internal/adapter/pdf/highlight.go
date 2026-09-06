package pdf

import (
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
)

// A word ends where the printing leaves a gap. The gaps are fractions of the
// height of the line, so they hold for a page set in any size.
const (
	// sameLine is how much of their height two characters of one line share.
	sameLine = 0.5
	// wordGap is the distance a space stands in, as a fraction of that height.
	wordGap = 0.25
)

// Highlights is where the words of some pages of the document sit: one box a
// word, at the offset in Text where that word begins.
//
// The pages wanted are given by their index, and only those are read. The
// boxes come back ascending by offset.
//
// A page whose text layer says nothing gives no boxes, and a scan gives none at
// all.
func (b *Book) Highlights(raw []byte, pages []int) ([]highlight.Box, error) {
	wanted := ordered(pages, len(b.Pages))
	if len(wanted) == 0 {
		return nil, nil
	}

	doc, err := open(raw)
	if err != nil {
		return nil, err
	}
	defer doc.close()

	var boxes []highlight.Box
	for _, page := range wanted {
		boxes = append(boxes, doc.words(page, b.Pages[page].Offset)...)
	}
	return boxes, nil
}

// ordered is the pages asked for, each of them once and in reading order. A
// page the document does not have is not a page.
func ordered(pages []int, most int) []int {
	seen := make(map[int]bool, len(pages))
	out := make([]int, 0, len(pages))
	for _, page := range pages {
		if page < 0 || page >= most || seen[page] {
			continue
		}
		seen[page] = true
		out = append(out, page)
	}
	sort.Ints(out)
	return out
}

// words is where the words of one page sit, as boxes in the document's text.
// offset is where that page's own text begins in it.
//
// A page that cannot be measured, or cannot be read, says nothing about where
// its words are.
func (d *document) words(index, offset int) []highlight.Box {
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}}
	sheet, ok := d.paper(page)
	if !ok {
		return nil
	}
	read, err := d.worker.GetPageTextStructured(&requests.GetPageTextStructured{
		Page: page,
		Mode: requests.GetPageTextStructuredModeChars,
	})
	if err != nil {
		return nil
	}

	var boxes []highlight.Box
	// The word being read: where it covers the page, where it begins in the
	// text, and how far it has got.
	var word responses.CharPosition
	start, length := 0, 0
	done := func() {
		if length > 0 {
			boxes = append(boxes, sheet.box(index, start, length, word))
		}
		length = 0
	}

	at := offset
	for _, char := range read.Chars {
		text := char.Text
		if text == "" {
			continue
		}
		mark := char.PointPosition
		if strings.TrimSpace(text) == "" || mark.Right <= mark.Left || mark.Top <= mark.Bottom {
			// A space ends a word, and so does a character covering nothing.
			done()
			at += len(text)
			continue
		}
		if length > 0 && !joins(word, mark) {
			done()
		}
		if length == 0 {
			start, word = at, mark
		} else {
			word = widen(word, mark)
		}
		length += len(text)
		at += len(text)
	}
	done()
	return boxes
}

// joins says whether a character carries the word on: it stands on the same
// line, within a space of where the word has got to.
func joins(word, mark responses.CharPosition) bool {
	high := min(word.Top-word.Bottom, mark.Top-mark.Bottom)
	shared := min(word.Top, mark.Top) - max(word.Bottom, mark.Bottom)
	if high <= 0 || shared < sameLine*high {
		return false
	}
	return mark.Left-word.Right <= wordGap*high && mark.Right > word.Right
}

// widen is the word with one more character in it.
func widen(word, mark responses.CharPosition) responses.CharPosition {
	return responses.CharPosition{
		Left:   min(word.Left, mark.Left),
		Right:  max(word.Right, mark.Right),
		Top:    max(word.Top, mark.Top),
		Bottom: min(word.Bottom, mark.Bottom),
	}
}

// sheet is one page in the space its text is written in: how wide and how high
// it is, in points, and the quarter turns clockwise it is drawn with.
type sheet struct {
	width, height float64
	turn          int
}

// paper measures one page. The size the library reports is of the page as it is
// drawn, and a page drawn a quarter turn from the way its text is written is as
// wide as its text is high.
func (d *document) paper(page requests.Page) (sheet, bool) {
	size, err := d.worker.GetPageSize(&requests.GetPageSize{Page: page})
	if err != nil || size.Width <= 0 || size.Height <= 0 {
		return sheet{}, false
	}
	turned, err := d.worker.FPDFPage_GetRotation(&requests.FPDFPage_GetRotation{Page: page})
	if err != nil {
		return sheet{}, false
	}
	sheet := sheet{width: size.Width, height: size.Height, turn: int(turned.PageRotation)}
	if sheet.turn%2 == 1 {
		sheet.width, sheet.height = sheet.height, sheet.width
	}
	return sheet, true
}

// box is one word of a page, over the fraction of it the word covers.
func (p sheet) box(page, start, length int, word responses.CharPosition) highlight.Box {
	x0, y0 := p.drawn(word.Left, word.Top)
	x1, y1 := p.drawn(word.Right, word.Bottom)
	return highlight.Box{
		Page:    page,
		Stretch: highlight.Stretch{Start: start, Length: length},
		Rect: highlight.Rect{
			MinX: onPage(min(x0, x1)),
			MinY: onPage(min(y0, y1)),
			MaxX: onPage(max(x0, x1)),
			MaxY: onPage(max(y0, y1)),
		},
	}
}

// drawn is where a point of the page falls on the page as it is drawn: a
// fraction of it, measured from its top left corner.
//
// A page's text is written with the origin at the bottom left corner, and the
// page is drawn with the origin at the top left and turned by however much it
// asks to be.
func (p sheet) drawn(x, y float64) (float32, float32) {
	wide, high := p.width, p.height
	switch p.turn {
	case 1:
		x, y, wide, high = y, x, high, wide
	case 2:
		x = p.width - x
	case 3:
		x, y, wide, high = p.height-y, p.width-x, high, wide
	default:
		y = p.height - y
	}
	return float32(x / wide), float32(y / high)
}

// onPage holds a rectangle to the page it is on. A character set past the edge
// of the paper is drawn at the edge.
func onPage(f float32) float32 {
	return min(max(f, 0), 1)
}
