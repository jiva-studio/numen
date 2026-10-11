package pdf_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// paddingSize is a trailing run large enough that a copy of the file shows
// against the allocator's own movement.
const paddingSize = 64 << 20

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, fixture(t, name), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestADocumentReadByPathSaysWhatTheSameBytesSay(t *testing.T) {
	for _, name := range []string{tiny, outline, labels, turned, scan} {
		t.Run(name, func(t *testing.T) {
			want := read(t, name)
			got, err := pdf.ReadFile(copyFixture(t, name))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("read by path:\n%+v\nread from bytes:\n%+v", got, want)
			}
		})
	}
}

func TestADocumentOpenedByPathIsDrawnAsTheSameBytesAre(t *testing.T) {
	for _, name := range []string{tiny, turned, scan} {
		t.Run(name, func(t *testing.T) {
			fromBytes, err := pdf.Open(fixture(t, name))
			if err != nil {
				t.Fatal(err)
			}
			defer fromBytes.Close()
			fromPath, err := pdf.OpenFile(copyFixture(t, name))
			if err != nil {
				t.Fatal(err)
			}
			defer fromPath.Close()

			if fromPath.Pages() != fromBytes.Pages() {
				t.Fatalf("%d pages, want %d", fromPath.Pages(), fromBytes.Pages())
			}
			for page := range fromBytes.Pages() {
				want, err := fromBytes.Image(page, 72)
				if err != nil {
					t.Fatal(err)
				}
				got, err := fromPath.Image(page, 72)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got.(*image.RGBA).Pix, want.(*image.RGBA).Pix) {
					t.Errorf("page %d is drawn differently from a path", page)
				}
			}
		})
	}
}

func TestADocumentOpenedByPathKeepsNoCopyOfTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "padded.pdf")
	raw := append(fixture(t, tiny), bytes.Repeat([]byte("%padding\n"), paddingSize/len("%padding\n"))...)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	size := uint64(len(raw))
	raw = nil

	// A worker is started before the first reading, so the baseline holds it.
	warm, err := pdf.OpenFile(copyFixture(t, tiny))
	if err != nil {
		t.Fatal(err)
	}
	warm.Close()

	runtime.GC()
	var before, during runtime.MemStats
	runtime.ReadMemStats(&before)
	held, err := pdf.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	runtime.GC()
	runtime.ReadMemStats(&during)

	if during.HeapAlloc > before.HeapAlloc && during.HeapAlloc-before.HeapAlloc > size/2 {
		t.Errorf("the open document retains %d bytes of a %d byte file", during.HeapAlloc-before.HeapAlloc, size)
	}
	if held.Pages() != 2 {
		t.Errorf("%d pages, want 2", held.Pages())
	}
}

func TestAFileThatIsMissingIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone.pdf")
	if _, err := pdf.ReadFile(missing); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("reading a missing file gave %v", err)
	}
	if _, err := pdf.OpenFile(missing); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("opening a missing file gave %v", err)
	}
}

func TestAFileThatIsNotADocumentIsRefusedByPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), prose)
	if err := os.WriteFile(path, fixture(t, prose), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := pdf.ReadFile(path); !errors.Is(err, pdf.ErrNotPDF) {
		t.Errorf("reading prose gave %v", err)
	}
	empty := filepath.Join(t.TempDir(), "empty.pdf")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := pdf.OpenFile(empty); !errors.Is(err, pdf.ErrNotPDF) {
		t.Errorf("opening an empty file gave %v", err)
	}
}

func TestAFileCanBeRemovedOnceItsDocumentIsClosed(t *testing.T) {
	path := copyFixture(t, tiny)
	held, err := pdf.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	held.Close()
	if err := os.Remove(path); err != nil {
		t.Errorf("the closed document still holds its file: %v", err)
	}

	path = copyFixture(t, tiny)
	if _, err := pdf.ReadFile(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Errorf("a file read still holds its file: %v", err)
	}

	path = copyFixture(t, prose)
	if _, err := pdf.ReadFile(path); err == nil {
		t.Fatal("prose was read")
	}
	if err := os.Remove(path); err != nil {
		t.Errorf("a file refused still holds its file: %v", err)
	}
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
			want, err := pdf.Open(fixture(t, name))
			if err != nil {
				t.Fatal(err)
			}
			defer want.Close()
			file := openCounted(t, copyFixture(t, name))
			drawn, err := pdf.Documents{}.Draw(t.Context(), file)
			if err != nil {
				t.Fatal(err)
			}
			if drawn.Pages() != want.Pages() {
				t.Fatalf("%d pages, want %d", drawn.Pages(), want.Pages())
			}
			for page := range want.Pages() {
				expected, err := want.Image(page, 72)
				if err != nil {
					t.Fatal(err)
				}
				got, err := drawn.Image(page, 72)
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
			_, err := pdf.Documents{}.Draw(t.Context(), file)
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
	if _, err := (pdf.Documents{}).Draw(ctx, file); !errors.Is(err, context.Canceled) {
		t.Errorf("drawing gave %v", err)
	}
	if file.closes != 1 {
		t.Errorf("the file was closed %d times, want 1", file.closes)
	}
}
