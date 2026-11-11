package document

import (
	"errors"
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/responses"
)

type stubPool struct {
	pdfium.Pool
	timeout  time.Duration
	instance pdfium.Pdfium
	err      error
}

func (p *stubPool) GetInstance(timeout time.Duration) (pdfium.Pdfium, error) {
	p.timeout = timeout
	return p.instance, p.err
}

type stubWorker struct {
	pdfium.Pdfium
	openReq     *requests.OpenDocument
	openErr     error
	closeErr    error
	renderReq   *requests.RenderPageInDPI
	renderImage image.Image
	renderErr   error
}

func (w *stubWorker) OpenDocument(req *requests.OpenDocument) (*responses.OpenDocument, error) {
	w.openReq = req
	if w.openErr != nil {
		return nil, w.openErr
	}
	return &responses.OpenDocument{Document: references.FPDF_DOCUMENT("doc")}, nil
}

func (w *stubWorker) RenderPageInDPI(req *requests.RenderPageInDPI) (*responses.RenderPageInDPI, error) {
	w.renderReq = req
	if w.renderErr != nil {
		return nil, w.renderErr
	}
	return &responses.RenderPageInDPI{
		Result: responses.RenderPage{
			Page: 0, PointToPixelRatio: 1,
			RenderedImage: w.renderImage, Width: 1, Height: 1, HasTransparency: false,
		},
		CleanupFunc: func() {},
	}, nil
}

func (w *stubWorker) Close() error {
	return w.closeErr
}

func TestGetExtensionsAndRuntimeConfig(t *testing.T) {
	exts := GetExtensions()
	if !slices.Contains(exts, ".pdf") || !slices.Contains(exts, ".png") {
		t.Fatalf("GetExtensions() = %v", exts)
	}
	reader := NewReader()
	if cfg := reader.getRuntimeConfig(); cfg != nil {
		t.Fatalf("getRuntimeConfig() = %v, want nil", cfg)
	}
}

func TestOpenPDFPropagatesWorkerAndDocumentErrors(t *testing.T) {
	poolErr := errors.New("pool error")
	reader := NewReader()
	reader.workers = &stubPool{Pool: nil, timeout: 0, instance: nil, err: poolErr}
	if _, err := reader.Open("scan.pdf", []byte("x")); !errors.Is(err, poolErr) {
		t.Fatalf("Open with failing pool returned %v, want %v", err, poolErr)
	}

	openErr := errors.New("bad pdf")
	closeErr := errors.New("close failed")
	worker := &stubWorker{
		Pdfium: nil, openReq: nil, openErr: openErr,
		closeErr: closeErr, renderReq: nil, renderImage: nil, renderErr: nil,
	}
	pool := &stubPool{Pool: nil, timeout: 0, instance: worker, err: nil}
	reader = NewReader()
	reader.workers = pool
	_, err := reader.Open("scan.pdf", []byte("x"))
	if !errors.Is(err, ErrUnreadable) || !errors.Is(err, openErr) || !errors.Is(err, closeErr) {
		t.Fatalf("Open with failing OpenDocument returned %v", err)
	}
	if pool.timeout != 2*time.Minute {
		t.Fatalf("GetInstance timeout = %v, want %v", pool.timeout, 2*time.Minute)
	}
	if worker.openReq == nil || worker.openReq.FileReaderSize != 0 {
		t.Fatalf("OpenDocument FileReaderSize = %v, want 0", worker.openReq)
	}
	if strings.Contains(err.Error(), "the file is empty") {
		t.Fatalf("1-byte input reported as empty: %v", err)
	}
}

func TestPDFRenderAndCloseBehavior(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	wantColor := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	src.SetRGBA(0, 0, wantColor)

	closeErr := errors.New("worker close error")
	worker := &stubWorker{
		Pdfium: nil, openReq: nil, openErr: nil,
		closeErr: closeErr, renderReq: nil, renderImage: src, renderErr: nil,
	}
	doc := &pdf{worker: worker, ref: "doc", pages: 2}

	if _, err := doc.RenderPage(-1, 72); err == nil || !strings.Contains(err.Error(), "pdf: page -1 of 2") {
		t.Fatalf("RenderPage(-1) = %v, want bounds error", err)
	}
	if _, err := doc.RenderPage(2, 72); err == nil || !strings.Contains(err.Error(), "pdf: page 2 of 2") {
		t.Fatalf("RenderPage(2) = %v, want bounds error", err)
	}

	rendered, err := doc.RenderPage(0, 72)
	if err != nil {
		t.Fatalf("RenderPage(0) failed: %v", err)
	}
	if worker.renderReq == nil || worker.renderReq.RenderFlags != 0 {
		t.Fatalf("RenderPageInDPI RenderFlags = %v, want 0", worker.renderReq)
	}
	rgba, ok := rendered.(*image.RGBA)
	if !ok || rgba.RGBAAt(0, 0) != wantColor {
		t.Fatalf("rendered pixel = %v, want %v", rendered.At(0, 0), wantColor)
	}

	worker.renderImage = image.NewGray(image.Rect(0, 0, 1, 1))
	if _, err := doc.RenderPage(1, 72); err == nil || !strings.Contains(err.Error(), "was drawn as") {
		t.Fatalf("RenderPage with non-RGBA image = %v, want type error", err)
	}

	renderErr := errors.New("draw failed")
	worker.renderErr = renderErr
	if _, err := doc.RenderPage(1, 72); !errors.Is(err, renderErr) {
		t.Fatalf("RenderPage with failing worker = %v, want %v", err, renderErr)
	}

	if err := doc.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close() = %v, want %v", err, closeErr)
	}
}
