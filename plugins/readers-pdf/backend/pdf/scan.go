package pdf

import (
	"fmt"
	"image"
	"io"

	"github.com/klippa-app/go-pdfium/requests"
)

// Scan renders raster images of document pages for recognition.
// It retains an underlying PDF worker until Close is called.
type Scan struct {
	doc *document
}

// OpenReader opens a PDF for page rendering, reading from the file as the
// library requests. The scan owns the file and closes it with the scan, or
// immediately if the document does not open.
func OpenReader(file io.ReadSeekCloser) (*Scan, error) {
	doc, err := openReader(file)
	if err != nil {
		return nil, err
	}
	return &Scan{doc: doc}, nil
}

// OpenBytes opens a PDF from raw bytes for page rendering.
func OpenBytes(raw []byte) (*Scan, error) {
	doc, err := openBytes(raw)
	if err != nil {
		return nil, err
	}
	return &Scan{doc: doc}, nil
}

func (s *Scan) Close() { s.doc.close() }

// GetPageCount is how many pages the document has.
func (s *Scan) GetPageCount() int { return s.doc.pages }

// GetPageSize returns the width and height at which one page is rendered, in
// the page's own units. Rendering at a given width derives its resolution from
// this size, and the document returns it without rendering.
//
// The page is rendered rotated by its own rotation, so a page whose text is
// written a quarter turn from the rendered orientation is as wide as its text
// is high, and the size reflects that.
//
// The page is not loaded to measure it: a window laying out a book requests the
// size of every page before rendering any of them, and loading hundreds of
// pages takes seconds.
func (s *Scan) GetPageSize(index int) (width, height float64, err error) {
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
	width, height, err := s.GetPageSize(index)
	// gomutants:disable-next-line BRANCH_IF reason="page dimension lookup failure stops rendering"
	if err != nil {
		return nil, err
	}
	// gomutants:disable-next-line ARITHMETIC_BASE reason="raster pixel count computes area scaled by DPI over point resolution"
	if pixels := width * height * float64(dpi) * float64(dpi) / (pointsPerInch * pointsPerInch); pixels > mostPixels {
		return nil, fmt.Errorf("pdf: page %d cannot be drawn at %d DPI: it is %.0f pixels, and %d is the most",
			index, dpi, pixels, mostPixels)
	}
	rendered, err := s.doc.worker.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI:  dpi,
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: s.doc.ref, Index: index}},
	})
	// gomutants:disable-next-line BRANCH_IF reason="rendering failure stops page rasterization"
	if err != nil {
		return nil, err
	}
	// The library returns an image over memory it owns and a function that
	// releases that memory. The pixels are copied out because the rendered
	// image outlives the call that rendered it. A page is rendered in color
	// unless a format was requested, and none is.
	page, isColour := rendered.Result.RenderedImage.(*image.RGBA)
	// gomutants:disable-next-line BRANCH_IF reason="non-RGBA render output is rejected"
	if !isColour {
		rendered.Cleanup()
		return nil, fmt.Errorf("pdf: page %d was drawn as %T", index, rendered.Result.RenderedImage)
	}
	out := image.NewRGBA(page.Bounds())
	copy(out.Pix, page.Pix)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="worker render buffer is released after copying pixels"
	rendered.Cleanup()
	return out, nil
}

func recoverPanic[T any](what string, answer *T, err *error) {
	if raised := recover(); raised != nil {
		var none T
		*answer, *err = none, fmt.Errorf("%s raised %v", what, raised)
	}
}
