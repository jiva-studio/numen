package document_test

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"io"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/epub"
)

// leaf is a page of a scan: its size, and whether the size can be read.
type leaf struct {
	wide, high float64
	unmeasured bool
}

// scan is a document with the pages it is given, drawn as blank sheets.
type scan struct{ leaves []leaf }

func (s scan) GetPageCount() int { return len(s.leaves) }

func (s scan) GetPageSize(index int) (wide, high float64, err error) {
	if s.leaves[index].unmeasured {
		return 0, 0, io.ErrUnexpectedEOF
	}
	return s.leaves[index].wide, s.leaves[index].high, nil
}

func (s scan) Render(_, width int) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, width, width)), nil
}

// shelf holds the one scan it is given open, and counts the files it opens.
type shelf struct {
	scan   scan
	opened int
	mu     sync.Mutex
}

func (s *shelf) UseDocument(
	ctx context.Context, _ document.Fingerprint,
	file func(context.Context) (io.ReadSeekCloser, error),
	use func(document.Scan) error,
) error {
	opened, err := file(ctx)
	if err != nil {
		return err
	}
	defer opened.Close()
	s.mu.Lock()
	s.opened++
	s.mu.Unlock()
	return use(s.scan)
}

// drawn holds the pages by key, and counts the drawings.
type drawn struct {
	pages map[document.PageKey][]byte
	count int
}

func (d *drawn) GetPage(_ context.Context, key document.PageKey, draw func() ([]byte, error)) ([]byte, error) {
	if body, held := d.pages[key]; held {
		return body, nil
	}
	body, err := draw()
	if err != nil {
		return nil, err
	}
	if d.pages == nil {
		d.pages = map[document.PageKey][]byte{}
	}
	d.pages[key] = body
	d.count++
	return body, nil
}

// kept reads every book it is asked for.
type kept struct{}

func (kept) GetBook(_ context.Context, _ document.Fingerprint, read func() (*epub.Book, error)) (*epub.Book, error) {
	return read()
}

// vaultOf is a vault holding the files it is given.
func vaultOf(t *testing.T, files map[string]string) domain.Vault {
	t.Helper()
	return testsupport.NewVault(t, files)
}

var readers = filesystem.VaultReaders{}

// newArchive is a zip holding the entries it is given.
func newArchive(t *testing.T, entries map[string]string) string {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for name, body := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

const (
	plate = "\x89PNG\r\n\x1a\npixels"
	cover = `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
)

// bookOf is an EPUB with one chapter, a picture, and a cover that is a document.
func bookOf(t *testing.T) string {
	t.Helper()
	return newArchive(t, map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>A Reflowed Book</dc:title>
    <dc:identifier id="id">reflowed</dc:identifier>
  </metadata>
  <manifest>
    <item id="a" href="first.xhtml" media-type="application/xhtml+xml"/>
    <item id="c" href="pictures/plate.png" media-type="image/png"/>
    <item id="d" href="pictures/cover.svg" media-type="image/svg+xml"/>
  </manifest>
  <spine><itemref idref="a"/></spine>
</package>`,
		"OEBPS/first.xhtml":        `<html><body><h1>The First Part</h1><p>Alpha omega.</p></body></html>`,
		"OEBPS/pictures/plate.png": plate,
		"OEBPS/pictures/cover.svg": cover,
	})
}
