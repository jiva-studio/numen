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

// Label is what the document calls one page.
func (s *Scan) Label(index int) string { return s.doc.label(index) }

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
