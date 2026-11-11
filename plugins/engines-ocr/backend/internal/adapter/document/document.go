// Package document opens the files the plugin recognizes, a scanned page or a PDF, and draws their
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
	"github.com/tetratelabs/wazero"
	_ "golang.org/x/image/webp" // a scan saved as WebP

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

// GetExtensions returns the extensions, with the dot, of the files the plugin recognizes.
func GetExtensions() []string {
	return []string{".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp"}
}

// ErrUnreadable is a file that opens as no document the plugin reads.
var ErrUnreadable = errors.New("the file opens as no document")

// Reader opens documents. It starts one PDFium worker with the first PDF and keeps it for its
// life.
type Reader struct {
	mu               sync.Mutex
	workers          pdfium.Pool
	compilationCache wazero.CompilationCache
}

// NewReader returns a reader that has started no worker.
func NewReader() *Reader {
	return &Reader{mu: sync.Mutex{}, workers: nil, compilationCache: nil}
}

// Open opens the bytes of the file at the path, by its extension.
func (r *Reader) Open(filePath string, raw []byte) (recognition.Document, error) {
	if strings.EqualFold(path.Ext(filePath), ".pdf") {
		return r.openPDF(raw)
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
func (picture) Close() error       { return nil }

func (p picture) RenderPage(index, _ int) (image.Image, error) {
	if index != 0 {
		return nil, fmt.Errorf("page %d of an image of one page", index)
	}
	return p.drawing, nil
}

// waitForWorker bounds the wait for the worker, so a worker wedged inside the library is a failure.
const waitForWorker = 2 * time.Minute

func (r *Reader) getWorker() (pdfium.Pdfium, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.workers == nil {
		started, err := webassembly.Init(webassembly.Config{
			// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Worker pool idle and total capacity limits are internal to webassembly.Init."
			Context: nil, MinIdle: 0, MaxIdle: 0, MaxTotal: 1, WASM: nil, FSConfig: nil,
			RuntimeConfig: r.getRuntimeConfig(), Stdout: nil, Stderr: nil, RandomSource: nil, ReuseWorkers: false,
		})
		// gomutants:disable-next-line BRANCH_IF reason="webassembly.Init uses the embedded PDFium binary and fixed configuration."
		if err != nil {
			// gomutants:disable-next-line RETURN_ERROR_NIL reason="webassembly.Init uses the embedded PDFium binary and fixed configuration."
			return nil, err
		}
		r.workers = started
	}
	return r.workers.GetInstance(waitForWorker)
}

// pdf is a PDF open on the worker, which it holds until it is closed.
type pdf struct {
	worker pdfium.Pdfium
	ref    references.FPDF_DOCUMENT
	pages  int
}

func (r *Reader) openPDF(raw []byte) (*pdf, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: the file is empty", ErrUnreadable)
	}
	worker, err := r.getWorker()
	if err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	opened, err := worker.OpenDocument(&requests.OpenDocument{
		File: &raw, FilePath: nil, FileReader: nil, FileReaderSize: 0, Password: nil,
	})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%w: %w", ErrUnreadable, err), worker.Close())
	}
	count, err := worker.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: opened.Document})
	// gomutants:disable-next-line BRANCH_IF reason="FPDF_GetPageCount only fails when the document handle is invalid, which OpenDocument already validated."
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%w: %w", ErrUnreadable, err), worker.Close())
	}
	return &pdf{worker: worker, ref: opened.Document, pages: count.PageCount}, nil
}

func (d *pdf) GetPageCount() int { return d.pages }

// Close gives the worker back. Closing the instance closes the document with it.
func (d *pdf) Close() error { return d.worker.Close() }

// HasTextLayer reports whether any page carries text that is not blank. A page whose layer cannot be
// read carries none.
func (d *pdf) HasTextLayer() bool {
	for index := range d.pages {
		page, err := d.worker.GetPageText(&requests.GetPageText{
			Page: requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}, ByReference: nil},
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
		DPI:         dpi,
		Page:        requests.Page{ByIndex: &requests.PageByIndex{Document: d.ref, Index: index}, ByReference: nil},
		RenderFlags: 0,
		RenderForm:  false,
		Document:    nil,
		Crop:        nil,
		ImageFormat: "",
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
