package testsupport

import (
	"bytes"
	"context"
	"image"
	"io"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// Documents is a page renderer for tests: a file that opens with the PDF marker is a document of
// one blank page.
type Documents struct{}

type dummyOpenDocument struct {
	file io.Closer
}

func (d dummyOpenDocument) GetPageCount() int {
	return 1
}

func (d dummyOpenDocument) RenderPage(_, _ int) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
}

func (d dummyOpenDocument) Close() {
	if d.file != nil {
		_ = d.file.Close()
	}
}

func (Documents) OpenDocument(_ context.Context, file io.ReadSeekCloser) (sourceport.OpenDocument, error) {
	if file != nil {
		buf := make([]byte, 5)
		n, _ := io.ReadFull(file, buf)
		if n < 5 || !bytes.HasPrefix(buf, []byte("%PDF-")) {
			_ = file.Close()
			return nil, sourceport.ErrNotADocument
		}
		_, _ = file.Seek(0, io.SeekStart)
	}
	return dummyOpenDocument{file: file}, nil
}
