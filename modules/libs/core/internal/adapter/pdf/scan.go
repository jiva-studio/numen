package pdf

import (
	"fmt"
	"image"

	"github.com/klippa-app/go-pdfium/requests"
)

// A Scan is a document held open so that its pages can be looked at rather than
// read: what a recogniser is given is the picture of a page, because a scan has
// no text to take out.
//
// It holds one of the library's workers for as long as it is open, so it is
// closed as soon as the document is done with.
type Scan struct {
	doc *document
}

// Open holds a document open for its pages to be drawn.
func Open(raw []byte) (*Scan, error) {
	doc, err := open(raw)
	if err != nil {
		return nil, err
	}
	return &Scan{doc: doc}, nil
}

func (s *Scan) Close() { s.doc.close() }

// Pages is how many pages the document has.
func (s *Scan) Pages() int { return s.doc.pages }

// Size is how wide and how high one page is drawn, in the page's own units. A
// drawing at a width works its resolution back from it, and the document
// answers without anything being drawn.
//
// The page is drawn turned by however much it asks to be, so a page whose text
// is written a quarter turn from the way it is drawn is drawn as wide as its
// text is high, and it is answered so.
//
// The page is not loaded to be measured. A window laying out a book asks for
// every page's size before it has drawn any of them, and loading five hundred
// pages to measure them is seconds before anything is on the screen.
func (s *Scan) Size(index int) (wide, high float64, err error) {
	if index < 0 || index >= s.doc.pages {
		return 0, 0, fmt.Errorf("pdf: page %d of %d", index, s.doc.pages)
	}
	size, err := s.doc.worker.FPDF_GetPageSizeByIndexF(&requests.FPDF_GetPageSizeByIndexF{
		Document: s.doc.ref, Index: index,
	})
	if err != nil || size.Size.Width <= 0 || size.Size.Height <= 0 {
		return 0, 0, fmt.Errorf("pdf: page %d cannot be measured", index)
	}
	return float64(size.Size.Width), float64(size.Size.Height), nil
}

// Image is one page drawn at the given resolution.
//
// What resolution to draw at is a setting, and it decides what a model sees: a
// page drawn too small loses the marks over its letters, and one drawn too large
// is read no better and costs the square of the difference.
func (s *Scan) Image(index, dpi int) (image.Image, error) {
	if index < 0 || index >= s.doc.pages {
		return nil, fmt.Errorf("pdf: page %d of %d", index, s.doc.pages)
	}
	if dpi <= 0 {
		dpi = 300
	}
	drawn, err := s.doc.worker.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI:  dpi,
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: s.doc.ref, Index: index}},
	})
	if err != nil {
		return nil, err
	}
	// The library hands back an image over memory it owns and a way to give that
	// memory back. The pixels are copied out, because what is drawn outlives the
	// call that drew it.
	page := drawn.Result.Image
	out := image.NewRGBA(page.Bounds())
	copy(out.Pix, page.Pix)
	drawn.Cleanup()
	return out, nil
}
