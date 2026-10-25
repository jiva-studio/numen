package document_test

import (
	"context"
	"image"
	"io"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/document"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// leaf is a page of a scan: its size, and whether the size can be read.
type leaf struct {
	width, height float64
	unmeasured    bool
}

// scan is a document with the pages it is given, drawn as blank sheets.
type scan struct{ leaves []leaf }

func (s scan) GetPageCount() int { return len(s.leaves) }

func (s scan) GetPageSize(index int) (width, height float64, err error) {
	if s.leaves[index].unmeasured {
		return 0, 0, io.ErrUnexpectedEOF
	}
	return s.leaves[index].width, s.leaves[index].height, nil
}

func (s scan) Render(_, width int) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, width, width)), nil
}

// fakeDocuments holds the test scan open and counts file open operations.
type fakeDocuments struct {
	scan   scan
	opened int
	mu     sync.Mutex
}

func (s *fakeDocuments) UseDocument(
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
	if body, target := d.pages[key]; target {
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

// vaultOf is a vault holding the files it is given.
func vaultOf(t *testing.T, files map[string]string) domain.Vault {
	t.Helper()
	return testsupport.NewVault(t, files)
}

var readers = vaultreader.Readers{}

const (
	plate = "\x89PNG\r\n\x1a\npixels"
	cover = `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
)

// archive is the entries of a book, and the requested fingerprints.
type archive struct {
	entries   map[string][]byte
	failure   error
	requested []domain.Fingerprint
}

func (a *archive) ReadEntry(_ context.Context, mark domain.Fingerprint, entry string) ([]byte, error) {
	a.requested = append(a.requested, mark)
	if a.failure != nil {
		return nil, a.failure
	}
	body, target := a.entries[entry]
	if !target {
		return nil, document.ErrNoEntry
	}
	return body, nil
}

// newBook is an archive with a picture and a cover that is a document.
func newBook() *archive {
	return &archive{entries: map[string][]byte{
		"OEBPS/pictures/plate.png": []byte(plate),
		"OEBPS/pictures/cover.svg": []byte(cover),
	}}
}

func (a *archive) GetBook(_ context.Context, path string) (domain.BookShape, error) {
	if a.failure != nil {
		return domain.BookShape{}, a.failure
	}
	return domain.BookShape{Title: path, Sections: []domain.BookSection{{Path: "OEBPS/first.xhtml", Length: 5}}}, nil
}

func (a *archive) ReadBookMarkup(_ context.Context, _, section string, baseline *domain.Fingerprint) (string, error) {
	if baseline != nil {
		a.requested = append(a.requested, *baseline)
	}
	if a.failure != nil {
		return "", a.failure
	}
	body, target := a.entries[section]
	if !target {
		return "", document.ErrNoEntry
	}
	return string(body), nil
}
