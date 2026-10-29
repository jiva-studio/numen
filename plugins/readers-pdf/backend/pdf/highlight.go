package pdf

import (
	"sort"
	"strings"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
)

// A Rect is a place on a page, in fractions of it.
type Rect struct {
	MinX, MinY, MaxX, MaxY float32
}

// A ByteRange is a half-open range of byte offsets in the text.
type ByteRange struct {
	Start, End int
}

// A HighlightBox is one run of prose and where it was read: the page it is on,
// the run of bytes in the text, and the rectangle it covers.
type HighlightBox struct {
	Page int
	ByteRange
	Rect
}

// A word ends where the printing leaves a gap. The gaps are fractions of the
// height of the line, so they hold for a page set in any size.
const (
	// sameLine is how much of their height two characters of one line share.
	sameLine = 0.5
	wordGap  = 0.25
)

// Highlights returns the locations of the words of some pages of the document:
// one box per word, at the offset in Text where that word begins.
//
// The requested pages are given by index, and only those are read. The boxes
// are returned in ascending offset order.
//
// A page whose text layer is empty yields no boxes, and a scan yields none at
// all.
func (b *Book) Highlights(raw []byte, pages []int) ([]HighlightBox, error) {
	wanted := getOrderedPages(pages, len(b.Pages))
	if len(wanted) == 0 {
		return nil, nil
	}

	doc, err := openBytes(raw)
	if err != nil {
		return nil, err
	}
	defer doc.close()

	var boxes []HighlightBox
	for _, page := range wanted {
		boxes = append(boxes, doc.words(page, b.Pages[page].Offset)...)
	}
	return boxes, nil
}

// getOrderedPages returns the requested pages, each once and in reading order. A
// page the document does not have is dropped.
func getOrderedPages(pages []int, most int) []int {
	observed := make(map[int]bool, len(pages))
	out := make([]int, 0, len(pages))
	for _, page := range pages {
		if page < 0 || page >= most || observed[page] {
			continue
		}
		observed[page] = true
		out = append(out, page)
	}
	sort.Ints(out)
	return out
}

// words returns the locations of the words of one page, as boxes in the document's text.
// offset is where that page's own text begins in it.
//
// A page that cannot be measured, or cannot be read, yields no word locations.
func (d *document) words(index, offset int) []HighlightBox {
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}}
	sheet, ok := d.getSheet(page)
	// gomutants:disable-next-line BRANCH_IF reason="words aborts when page sheet cannot be retrieved"
	if !ok {
		return nil
	}
	read, err := d.worker.GetPageTextStructured(&requests.GetPageTextStructured{
		Page: page,
		Mode: requests.GetPageTextStructuredModeChars,
	})
	// gomutants:disable-next-line BRANCH_IF reason="words aborts when text characters cannot be retrieved"
	if err != nil {
		return nil
	}

	collector := &wordReading{sheet: sheet, index: index}
	pos := offset
	for _, char := range read.Chars {
		text := char.Text
		// gomutants:disable-next-line BRANCH_IF reason="empty characters are skipped"
		if text == "" {
			continue
		}
		collector.add(pos, len(text), char.PointPosition, strings.TrimSpace(text) == "")
		pos += len(text)
	}
	collector.finish()
	return collector.boxes
}

// wordReading is the words of a page read character by character: the boxes
// done, and the word being read, with where it covers the page, where it begins
// in the text, and how far it has got.
type wordReading struct {
	sheet  pageGeometry
	index  int
	boxes  []HighlightBox
	word   responses.CharPosition
	start  int
	length int
}

// add takes the next character, which begins at the given place in the text. A
// space ends a word, and so does a character covering nothing.
func (r *wordReading) add(pos, size int, mark responses.CharPosition, isSpace bool) {
	if isSpace || mark.Right <= mark.Left || mark.Top <= mark.Bottom {
		r.finish()
		return
	}
	if r.length > 0 && !isWordContinued(r.word, mark) {
		r.finish()
	}
	if r.length == 0 {
		r.start, r.word = pos, mark
	} else {
		r.word = extendWord(r.word, mark)
	}
	r.length += size
}

// finish ends the word being read, and keeps its box where it has any
// characters.
func (r *wordReading) finish() {
	if r.length > 0 {
		r.boxes = append(r.boxes, r.sheet.getWordBox(r.index, r.start, r.length, r.word))
	}
	r.length = 0
}

// isWordContinued reports whether a character continues the word: it is on
// the same line, within a space of where the word has reached.
func isWordContinued(word, mark responses.CharPosition) bool {
	high := min(word.Top-word.Bottom, mark.Top-mark.Bottom)
	shared := min(word.Top, mark.Top) - max(word.Bottom, mark.Bottom)
	if high <= 0 || shared < sameLine*high {
		return false
	}
	return mark.Left-word.Right <= wordGap*high && mark.Right > word.Right
}

// extendWord is the word with one more character in it.
func extendWord(word, mark responses.CharPosition) responses.CharPosition {
	return responses.CharPosition{
		Left:   min(word.Left, mark.Left),
		Right:  max(word.Right, mark.Right),
		Top:    max(word.Top, mark.Top),
		Bottom: min(word.Bottom, mark.Bottom),
	}
}

// pageGeometry is one page in the space its text is written in: how wide and how high
// it is, in points, and the quarter turns clockwise it is drawn with.
type pageGeometry struct {
	width, height float64
	rotation      int
}

// getSheet measures one page. The size the library reports is of the page as it is
// drawn, and a page drawn a quarter turn from the way its text is written is as
// wide as its text is high.
func (d *document) getSheet(page requests.Page) (pageGeometry, bool) {
	size, err := d.worker.GetPageSize(&requests.GetPageSize{Page: page})
	if err != nil || size.Width <= 0 || size.Height <= 0 {
		return pageGeometry{}, false
	}
	turned, err := d.worker.FPDFPage_GetRotation(&requests.FPDFPage_GetRotation{Page: page})
	// gomutants:disable-next-line BRANCH_IF reason="getSheet returns false when rotation cannot be retrieved"
	if err != nil {
		return pageGeometry{}, false
	}
	sheet := pageGeometry{width: size.Width, height: size.Height, rotation: int(turned.PageRotation)}
	if sheet.rotation%2 == 1 {
		sheet.width, sheet.height = sheet.height, sheet.width
	}
	return sheet, true
}

// getWordBox is one word of a page, over the fraction of it the word covers.
func (p pageGeometry) getWordBox(page, start, length int, word responses.CharPosition) HighlightBox {
	x0, y0 := p.getPagePoint(word.Left, word.Top)
	x1, y1 := p.getPagePoint(word.Right, word.Bottom)
	return HighlightBox{
		Page:      page,
		ByteRange: ByteRange{Start: start, End: start + length},
		Rect: Rect{
			MinX: clampToPage(min(x0, x1)),
			MinY: clampToPage(min(y0, y1)),
			MaxX: clampToPage(max(x0, x1)),
			MaxY: clampToPage(max(y0, y1)),
		},
	}
}

// getPagePoint returns where a point of the page falls on the page as rendered: a
// fraction of it, measured from its top left corner.
//
// A page's text is written with the origin at the bottom left corner, and the
// page is rendered with the origin at the top left and rotated by the page's
// own rotation.
func (p pageGeometry) getPagePoint(x, y float64) (float32, float32) {
	wide, high := p.width, p.height
	switch p.rotation {
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

// clampToPage holds a rectangle to the page it is on. A character set past the edge
// of the paper is drawn at the edge.
func clampToPage(f float32) float32 {
	return min(max(f, 0), 1)
}
