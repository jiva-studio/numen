// Package document opens the files the plugin recognises, a scanned page or a PDF, and draws their
// pages for the recogniser.
package document

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // a scan saved as GIF
	_ "image/jpeg" // a scan saved as JPEG
	_ "image/png"  // a scan saved as PNG
	"path"
	"strings"
	"sync"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	_ "golang.org/x/image/webp" // a scan saved as WebP
)

// Extensions are the files the plugin recognises, with the dot.
var Extensions = []string{".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp"}

// ErrUnreadable is a file that opens as no document the plugin reads.
var ErrUnreadable = errors.New("the file opens as no document")

// Document is one file open for its pages to be drawn.
type Document interface {
	GetPageCount() int
	// HasTextLayer reports whether any page carries text of its own.
	HasTextLayer() bool
	// RenderPage draws one page, counted from zero, at the resolution given.
	RenderPage(index, dpi int) (image.Image, error)
	Close()
}

// Open opens the bytes of the file at the path, by its extension.
func Open(filePath string, raw []byte) (Document, error) {
	if strings.EqualFold(path.Ext(filePath), ".pdf") {
		return openPDF(raw)
	}
	drawing, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnreadable, filePath, err)
	}
	return picture{drawing: drawing}, nil
}

// picture is a scan saved as an image: one page that is the image itself.
type picture struct {
	drawing image.Image
}

func (picture) GetPageCount() int  { return 1 }
func (picture) HasTextLayer() bool { return false }
func (picture) Close()             {}

func (p picture) RenderPage(index, _ int) (image.Image, error) {
	if index != 0 {
		return nil, fmt.Errorf("page %d of an image of one page", index)
	}
	return p.drawing, nil
}

// pool is the one PDFium worker of the process, started with the first PDF and kept for its life.
var pool struct {
	sync.Mutex
	workers pdfium.Pool
}

// waitForWorker bounds the wait for the worker, so a worker wedged inside the library is a failure.
const waitForWorker = 2 * time.Minute

func getWorker() (pdfium.Pdfium, error) {
	pool.Lock()
	defer pool.Unlock()
	if pool.workers == nil {
		started, err := webassembly.Init(webassembly.Config{MaxTotal: 1})
		if err != nil {
			return nil, err
		}
		pool.workers = started
	}
	return pool.workers.GetInstance(waitForWorker)
}

// pdf is a PDF open on the worker, which it holds until it is closed.
type pdf struct {
	worker pdfium.Pdfium
	ref    references.FPDF_DOCUMENT
	pages  int
}

func openPDF(raw []byte) (*pdf, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: the file is empty", ErrUnreadable)
	}
	worker, err := getWorker()
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	opened, err := worker.OpenDocument(&requests.OpenDocument{File: &raw})
	if err != nil {
		worker.Close()
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	count, err := worker.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: opened.Document})
	if err != nil {
		worker.Close()
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	return &pdf{worker: worker, ref: opened.Document, pages: count.PageCount}, nil
}

func (d *pdf) GetPageCount() int { return d.pages }

// Close gives the worker back. Closing the instance closes the document with it.
func (d *pdf) Close() { d.worker.Close() }

// HasTextLayer reports whether any page carries text that is not blank. A page whose layer cannot be
// read carries none.
func (d *pdf) HasTextLayer() bool {
	for index := range d.pages {
		page, err := d.worker.GetPageText(&requests.GetPageText{
			Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}},
		})
		if err == nil && strings.TrimSpace(page.Text) != "" {
			return true
		}
	}
	return false
}

func (d *pdf) RenderPage(index, dpi int) (image.Image, error) {
	if index < 0 || index >= d.pages {
		return nil, fmt.Errorf("pdf: page %d of %d", index, d.pages)
	}
	rendered, err := d.worker.RenderPageInDPI(&requests.RenderPageInDPI{
		DPI:  dpi,
		Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}},
	})
	if err != nil {
		return nil, fmt.Errorf("pdf: draw page %d: %w", index, err)
	}
	// The library draws over memory it owns, so the pixels are copied out before it is given back.
	defer rendered.Cleanup()
	page, isRGBA := rendered.Result.RenderedImage.(*image.RGBA)
	if !isRGBA {
		return nil, fmt.Errorf("pdf: page %d was drawn as %T", index, rendered.Result.RenderedImage)
	}
	out := image.NewRGBA(page.Bounds())
	copy(out.Pix, page.Pix)
	return out, nil
}
