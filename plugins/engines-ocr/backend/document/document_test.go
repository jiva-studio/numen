package document_test

import (
	"bytes"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/document"
)

func open(t *testing.T, path string, raw []byte) document.Document {
	t.Helper()
	doc, err := document.Open(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	cleanup.CloseAfter(t, doc)
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
	for path, raw := range map[string][]byte{
		"a.pdf": []byte("INVALID_PDF_HEADER"), "b.pdf": nil, "c.png": []byte("not an image"),
	} {
		if _, err := document.Open(path, raw); !errors.Is(err, document.ErrUnreadable) {
			t.Errorf("%s opened with %v", path, err)
		}
	}
}
