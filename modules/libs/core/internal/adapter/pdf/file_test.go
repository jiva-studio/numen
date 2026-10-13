package pdf_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, fixture(t, name), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// countedFile is a file that counts how often it is closed.
type countedFile struct {
	*os.File
	closes int
}

func (c *countedFile) Close() error {
	c.closes++
	return c.File.Close()
}

func openCounted(t *testing.T, path string) *countedFile {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return &countedFile{File: file}
}

func TestADocumentDrawnFromAFileIsDrawnAsTheSameBytesAre(t *testing.T) {
	for _, name := range []string{tiny, turned, scan} {
		t.Run(name, func(t *testing.T) {
			want, err := pdf.OpenReader(readerOf(fixture(t, name)))
			if err != nil {
				t.Fatal(err)
			}
			defer want.Close()
			file := openCounted(t, copyFixture(t, name))
			drawn, err := pdf.Documents{}.OpenDocument(t.Context(), file)
			if err != nil {
				t.Fatal(err)
			}
			if drawn.GetPageCount() != want.GetPageCount() {
				t.Fatalf("%d pages, want %d", drawn.GetPageCount(), want.GetPageCount())
			}
			for page := range want.GetPageCount() {
				expected, err := want.RenderPage(page, 72)
				if err != nil {
					t.Fatal(err)
				}
				got, err := drawn.RenderPage(page, 72)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got.(*image.RGBA).Pix, expected.(*image.RGBA).Pix) {
					t.Errorf("page %d is drawn differently from a file", page)
				}
			}
			if file.closes != 0 {
				t.Errorf("the file was closed %d times while the document was open", file.closes)
			}
			drawn.Close()
			if file.closes != 1 {
				t.Errorf("the file was closed %d times by closing the document, want 1", file.closes)
			}
		})
	}
}

func TestAFileThatDoesNotDrawIsRefusedAndClosed(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"empty": empty, "prose": copyFixture(t, prose)} {
		t.Run(name, func(t *testing.T) {
			file := openCounted(t, path)
			_, err := pdf.Documents{}.OpenDocument(t.Context(), file)
			if !errors.Is(err, port.ErrNotADocument) || !errors.Is(err, pdf.ErrNotPDF) {
				t.Errorf("drawing it gave %v", err)
			}
			if file.closes != 1 {
				t.Errorf("the file was closed %d times, want 1", file.closes)
			}
		})
	}
}

func TestAFileIsClosedWhenDrawingIsCancelled(t *testing.T) {
	file := openCounted(t, copyFixture(t, tiny))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := (pdf.Documents{}).OpenDocument(ctx, file); !errors.Is(err, context.Canceled) {
		t.Errorf("drawing gave %v", err)
	}
	if file.closes != 1 {
		t.Errorf("the file was closed %d times, want 1", file.closes)
	}
}

// bytesFile is a document held in memory, with a Close that has nothing to release.
type bytesFile struct{ *bytes.Reader }

func (bytesFile) Close() error { return nil }

func readerOf(raw []byte) bytesFile { return bytesFile{bytes.NewReader(raw)} }
