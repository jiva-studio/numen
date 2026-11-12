package pdf

import (
	"errors"
	"sort"
	"strings"

	"github.com/klippa-app/go-pdfium"
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

// Highlights returns the locations of the words of some pages of the document: one box per word, at the offset in Text where that word begins.
// Pages are given by index and only those are read. The boxes are returned in ascending offset order.
// A page whose text layer is empty yields no boxes, and a scan yields none at all.
func (b *Book) Highlights(workers *Workers, raw []byte, pages []int) (boxes []HighlightBox, err error) {
	wanted := getOrderedPages(pages, len(b.Pages))
	// gomutants:disable-next-line INTEGER_DECREMENT,BRANCH_IF reason="empty page selection returns immediately without opening a worker"
	if len(wanted) == 0 {
		return nil, nil
	}

	doc, err := workers.openBytes(raw)
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Highlights releases the worker and closes the document on return"
	defer func() { err = errors.Join(err, doc.Close()) }()

	for _, page := range wanted {
		boxes = append(boxes, doc.words(page, b.Pages[page].Offset)...)
	}
	return boxes, nil
}

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

func (d *document) words(index, offset int) []HighlightBox {
	page := requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}, ByReference: nil}
	sheet, ok := getSheet(pdfiumMeasurer{worker: d.worker}, page)
	// gomutants:disable-next-line BRANCH_IF reason="words aborts when page sheet cannot be retrieved"
	if !ok {
		return nil
	}
	read, err := d.worker.GetPageTextStructured(&requests.GetPageTextStructured{
		Page:                   page,
		Mode:                   requests.GetPageTextStructuredModeChars,
		CollectFontInformation: false,
		PixelPositions: requests.GetPageTextStructuredPixelPositions{
			Document:  "",
			Calculate: false,
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="PixelPositions dimensions are unused when Calculate is false"
			DPI: 0,
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="PixelPositions dimensions are unused when Calculate is false"
			Width: 0,
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="PixelPositions dimensions are unused when Calculate is false"
			Height: 0,
		},
	})
	// gomutants:disable-next-line BRANCH_IF reason="words aborts when text characters cannot be retrieved"
	if err != nil {
		return nil
	}

	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="wordReading initializes word bounding box and counters to zero"
	collector := &wordReading{sheet: sheet, index: index, boxes: nil, word: responses.CharPosition{Left: 0, Right: 0, Top: 0, Bottom: 0}, start: 0, length: 0}
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

func (r *wordReading) add(pos, size int, mark responses.CharPosition, isSpace bool) {
	if isSpace || mark.Right <= mark.Left || mark.Top <= mark.Bottom {
		r.finish()
		return
	}
	// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_BOUNDARY,CONDITIONALS_NEGATION,INTEGER_DECREMENT,INTEGER_INCREMENT,BRANCH_IF reason="finish is a no-op when length is zero"
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

func (r *wordReading) finish() {
	if r.length > 0 {
		r.boxes = append(r.boxes, r.sheet.getWordBox(r.index, r.start, r.length, r.word))
	}
	r.length = 0
}

func isWordContinued(word, mark responses.CharPosition) bool {
	high := min(word.Top-word.Bottom, mark.Top-mark.Bottom)
	shared := min(word.Top, mark.Top) - max(word.Bottom, mark.Bottom)
	if high <= 0 || shared < sameLine*high {
		return false
	}
	return mark.Left-word.Right <= wordGap*high && mark.Right > word.Right
}

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

// pageMeasurer is the part of the pdfium worker that measures a page.
type pageMeasurer interface {
	GetPageSize(request *requests.GetPageSize) (*responses.GetPageSize, error)
	GetPageRotation(request *requests.FPDFPage_GetRotation) (*responses.FPDFPage_GetRotation, error)
}

// pdfiumMeasurer measures pages with a pdfium worker.
type pdfiumMeasurer struct {
	worker pdfium.Pdfium
}

func (m pdfiumMeasurer) GetPageSize(request *requests.GetPageSize) (*responses.GetPageSize, error) {
	return m.worker.GetPageSize(request)
}

// GetPageRotation gets the rotation of a page in quarter turns.
func (m pdfiumMeasurer) GetPageRotation(request *requests.FPDFPage_GetRotation) (*responses.FPDFPage_GetRotation, error) {
	return m.worker.FPDFPage_GetRotation(request)
}

func getSheet(measurer pageMeasurer, page requests.Page) (pageGeometry, bool) {
	size, err := measurer.GetPageSize(&requests.GetPageSize{Page: page})
	if err != nil || size.Width <= 0 || size.Height <= 0 {
		// gomutants:disable-next-line RETURN_ZERO,INTEGER_DECREMENT,INTEGER_INCREMENT reason="zero pageGeometry fields are ignored when getSheet returns false"
		return pageGeometry{width: 0, height: 0, rotation: 0}, false
	}
	turned, err := measurer.GetPageRotation(&requests.FPDFPage_GetRotation{Page: page})
	// gomutants:disable-next-line BRANCH_IF reason="getSheet returns false when rotation cannot be retrieved"
	if err != nil {
		// gomutants:disable-next-line RETURN_ZERO,INTEGER_DECREMENT,INTEGER_INCREMENT reason="zero pageGeometry fields are ignored when getSheet returns false"
		return pageGeometry{width: 0, height: 0, rotation: 0}, false
	}
	sheet := pageGeometry{width: size.Width, height: size.Height, rotation: int(turned.PageRotation)}
	if sheet.rotation%2 == 1 {
		sheet.width, sheet.height = sheet.height, sheet.width
	}
	return sheet, true
}

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

func clampToPage(f float32) float32 {
	return min(max(f, 0), 1)
}
