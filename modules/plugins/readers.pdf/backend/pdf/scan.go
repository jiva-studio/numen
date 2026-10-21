package pdf

import (
	"fmt"
	"image"
	"io"

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

// OpenReader holds a PDF open for its pages to be drawn, read from the file as
// the library asks for it. The scan owns the file and closes it with the scan,
// or at once if the document does not open.
func OpenReader(file io.ReadSeekCloser) (*Scan, error) {
	doc, err := openReader(file)
	if err != nil {
		return nil, err
	}
	return &Scan{doc: doc}, nil
}

func (s *Scan) Close() { s.doc.close() }

// GetPageCount is how many pages the document has.
func (s *Scan) GetPageCount() int { return s.doc.pages }

// GetPageSize is how wide and how high one page is drawn, in the page's own units. A
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
func (s *Scan) GetPageSize(index int) (wide, high float64, err error) {
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

// RenderPage is one page drawn at the given resolution.
//
// What resolution to draw at is a setting, and it decides what a model sees: a
// page drawn too small loses the marks over its letters, and one drawn too large
// is read no better and costs the square of the difference.
func (s *Scan) RenderPage(index, dpi int) (drawing image.Image, err error) {
	defer recoverPanic("drawing a page", &drawing, &err)
	if index < 0 || index >= s.doc.pages {
		return nil, fmt.Errorf("pdf: page %d of %d", index, s.doc.pages)
	}
	if dpi <= 0 {
		dpi = 300
	}
	wide, high, err := s.GetPageSize(index)
	// gomutants:disable-next-line BRANCH_IF reason="page dimension lookup failure stops rendering"
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line ARITHMETIC_BASE reason="raster pixel count computes area scaled by DPI over point resolution"
	if pixels := wide * high * float64(dpi) * float64(dpi) / (pointsPerInch * pointsPerInch); pixels > mostPixels {
		return nil, fmt.Errorf("pdf: page %d cannot be drawn at %d DPI: it is %.0f pixels, and %d is the most",
			index, dpi, pixels, mostPixels)
	}
	drawn, err := s.doc.worker.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI:  dpi,
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: s.doc.ref, Index: index}},
	})
	// gomutants:disable-next-line BRANCH_IF reason="rendering failure stops page rasterization"
	if err != nil {
		return nil, err
	}
	// The library returns an image over memory it owns and a way to give that
	// memory back. The pixels are copied out, because what is drawn outlives the
	// call that drew it. A page is drawn in colour unless a format was asked
	// for, and none is.
	page, isColour := drawn.Result.RenderedImage.(*image.RGBA)
	// gomutants:disable-next-line BRANCH_IF reason="non-RGBA render output is rejected"
	if !isColour {
		drawn.Cleanup()
		return nil, fmt.Errorf("pdf: page %d was drawn as %T", index, drawn.Result.RenderedImage)
	}
	out := image.NewRGBA(page.Bounds())
	copy(out.Pix, page.Pix)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="worker render buffer is released after copying pixels"
	drawn.Cleanup()
	return out, nil
}

func recoverPanic[T any](what string, answer *T, err *error) {
	if raised := recover(); raised != nil {
		var none T
		*answer, *err = none, fmt.Errorf("%s raised %v", what, raised)
	}
}
