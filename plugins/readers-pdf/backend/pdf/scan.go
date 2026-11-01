package pdf

import (
	"fmt"
	"image"
	"io"
	"math"

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
func (w *Workers) OpenReader(file io.ReadSeekCloser) (*Scan, error) {
	doc, err := w.openReader(file)
	if err != nil {
		return nil, err
	}
	return &Scan{doc: doc}, nil
}

// OpenBytes opens a PDF from raw bytes for page rendering.
func (w *Workers) OpenBytes(raw []byte) (*Scan, error) {
	doc, err := w.openBytes(raw)
	if err != nil {
		return nil, err
	}
	return &Scan{doc: doc}, nil
}

// Close releases the document and its worker.
func (s *Scan) Close() error { return s.doc.Close() }

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
		DPI:         dpi,
		Page:        requests.Page{ByIndex: &requests.PageByIndex{Document: s.doc.ref, Index: index}, ByReference: nil},
		RenderFlags: 0,
		RenderForm:  false,
		Document:    nil,
		Crop:        nil,
		ImageFormat: "",
	})
	// gomutants:disable-next-line BRANCH_IF reason="rendering failure stops page rasterization"
	if err != nil {
		return nil, err
	}
	return copyRendered(index, rendered.Result.RenderedImage, rendered.Cleanup)
}

// RenderPageAtWidth is one page drawn width pixels across, as high as its shape makes it.
//
// A window draws a page at the width it shows it at, so a page shown small costs little and one
// shown large is sharp. A page that would have more than maxPixels pixels at that width is an
// error: GetFittingWidth is the widest width that fits.
func (s *Scan) RenderPageAtWidth(index, width, maxPixels int) (drawing image.Image, err error) {
	defer recoverPanic("drawing a page", &drawing, &err)
	if width <= 0 {
		return nil, fmt.Errorf("pdf: page %d cannot be drawn %d pixels across", index, width)
	}
	pageWidth, pageHeight, err := s.GetPageSize(index)
	if err != nil {
		return nil, err
	}
	height := getHeightAt(width, pageWidth, pageHeight)
	if width*height > maxPixels {
		return nil, fmt.Errorf("pdf: page %d cannot be drawn %d pixels across: it is %d pixels, and %d is the most",
			index, width, width*height, maxPixels)
	}
	rendered, err := s.doc.worker.RenderPageInPixels(&requests.RenderPageInPixels{
		Page:        requests.Page{ByIndex: &requests.PageByIndex{Document: s.doc.ref, Index: index}, ByReference: nil},
		Width:       width,
		Height:      height,
		RenderFlags: 0,
		RenderForm:  false,
		Document:    nil,
		Crop:        nil,
		ImageFormat: "",
	})
	if err != nil {
		return nil, err
	}
	return copyRendered(index, rendered.Result.RenderedImage, rendered.Cleanup)
}

// GetFittingWidth is the widest a page can be drawn with no more than maxPixels pixels, but never
// wider than want and never under one pixel.
func (s *Scan) GetFittingWidth(index, want, maxPixels int) (int, error) {
	pageWidth, pageHeight, err := s.GetPageSize(index)
	if err != nil {
		return 0, err
	}
	fitting := int(math.Floor(math.Sqrt(float64(maxPixels) * pageWidth / pageHeight)))
	// The height is rounded, so the area of the estimate can be a row over.
	for fitting > 1 && fitting*getHeightAt(fitting, pageWidth, pageHeight) > maxPixels {
		fitting--
	}
	return max(min(want, fitting), 1), nil
}

// getHeightAt is how high a page of that shape is drawn width pixels across.
func getHeightAt(width int, pageWidth, pageHeight float64) int {
	return max(int(math.Round(float64(width)*pageHeight/pageWidth)), 1)
}

// copyRendered copies the pixels of a rendered page out of the memory the library owns and
// releases that memory. The copy is made because the image outlives the call that rendered it.
// A page is rendered in color unless a format was requested, and none is.
func copyRendered(index int, rendered image.Image, release func()) (image.Image, error) {
	page, isColour := rendered.(*image.RGBA)
	// gomutants:disable-next-line BRANCH_IF reason="non-RGBA render output is rejected"
	if !isColour {
		release()
		return nil, fmt.Errorf("pdf: page %d was drawn as %T", index, rendered)
	}
	out := image.NewRGBA(page.Bounds())
	copy(out.Pix, page.Pix)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="worker render buffer is released after copying pixels"
	release()
	return out, nil
}

func recoverPanic[T any](what string, answer *T, err *error) {
	if raised := recover(); raised != nil {
		var none T
		*answer, *err = none, fmt.Errorf("%s raised %v", what, raised)
	}
}
