package pdf_test

import (
	"bytes"
	"errors"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
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
