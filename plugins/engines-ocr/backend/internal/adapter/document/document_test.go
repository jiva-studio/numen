package document_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
	"github.com/tetratelabs/wazero"
)

// cacheEnv names the folder the tests of this package compile PDFium into once.
const cacheEnv = "NUMEN_TEST_WASM_CACHE"

// TestMain gives every reader of the tests one folder to cache the compiled PDFium in, because each
// reader compiles it again otherwise and a run under the race detector takes minutes for each.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "numen-ocr-wasm-")
	if err != nil {
		os.Exit(1)
	}
	if err := os.Setenv(cacheEnv, dir); err != nil {
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil && code == 0 {
		code = 1
	}
	os.Exit(code)
}

func newCompilation(t *testing.T) wazero.CompilationCache {
	t.Helper()
	compilation := wasm.NewCompilationCache(os.Getenv(cacheEnv))
	t.Cleanup(func() {
		if err := compilation.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	return compilation
}

func newReader(t *testing.T) *document.Reader {
	t.Helper()
	reader := document.NewReader()
	reader.SetCompilationCache(newCompilation(t))
	return reader
}

func open(t *testing.T, path string, raw []byte) recognition.Document {
	t.Helper()
	doc, err := newReader(t).Open(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeAndCheck(t, doc) })
	return doc
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestScannedPDFHasNoTextLayer(t *testing.T) {
	doc := open(t, "books/scan.pdf", readFixture(t, "scan.pdf"))

	if doc.GetPageCount() != 2 || doc.HasTextLayer() {
		t.Fatalf("pages %d, text layer %v; want two pages and no layer", doc.GetPageCount(), doc.HasTextLayer())
	}
	page, err := doc.RenderPage(1, 72)
	if err != nil {
		t.Fatal(err)
	}
	if size := page.Bounds().Size(); size.X != 200 || size.Y != 200 {
		t.Errorf("a page of 200 points drawn at 72 DPI is %v", size)
	}
	if _, err := doc.RenderPage(2, 72); err == nil {
		t.Error("a page past the last was drawn")
	}
}

func TestPDFWithTextHasTextLayer(t *testing.T) {
	if doc := open(t, "books/tiny.PDF", readFixture(t, "tiny.pdf")); !doc.HasTextLayer() {
		t.Error("a PDF that carries text has no layer")
	}
}

func TestImageIsOnePage(t *testing.T) {
	var raw bytes.Buffer
	if err := png.Encode(&raw, image.NewGray(image.Rect(0, 0, 30, 20))); err != nil {
		t.Fatal(err)
	}
	doc := open(t, "scans/receipt.png", raw.Bytes())

	page, err := doc.RenderPage(0, 300)
	if err != nil || doc.GetPageCount() != 1 || doc.HasTextLayer() || page.Bounds().Dx() != 30 {
		t.Fatalf("pages %d, layer %v, page %v (%v)", doc.GetPageCount(), doc.HasTextLayer(), page, err)
	}
	if _, err := doc.RenderPage(1, 300); err == nil {
		t.Error("a second page of an image was drawn")
	}
}

func TestOpenRejectsInvalidBytes(t *testing.T) {
	reader := newReader(t)
	for path, raw := range map[string][]byte{
		"a.pdf": []byte("INVALID_PDF_HEADER"), "b.pdf": nil, "c.png": []byte("not an image"),
	} {
		_, err := reader.Open(path, raw)
		if !errors.Is(err, document.ErrUnreadable) {
			t.Errorf("%s opened with %v", path, err)
		}
		if path == "b.pdf" && (err == nil || !bytes.Contains([]byte(err.Error()), []byte("the file is empty"))) {
			t.Errorf("empty pdf error = %v, want 'the file is empty'", err)
		}
		if path == "c.png" && !errors.Is(err, image.ErrFormat) {
			t.Errorf("invalid image error = %v, want image.ErrFormat", err)
		}
	}
}

// closeAndCheck closes c and fails the test when closing returns an error.
func closeAndCheck(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}
