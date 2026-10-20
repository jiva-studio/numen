package testsupport

import (
	"bytes"
	"context"
	"image"
	"io"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// PDFBook represents a PDF document structure for testing.
type PDFBook struct {
	Text string
}

// Documents provides a stub implementation of document ports for testing.
type Documents struct{}

func (Documents) Read(_ context.Context, raw []byte) (domain.TextLayer, error) {
	if !bytes.HasPrefix(raw, []byte("%PDF-")) {
		return domain.TextLayer{}, port.ErrNotADocument
	}
	return domain.TextLayer{
		Text:  "Alpha Beta\nGamma Delta\n",
		Pages: []int{0, 11},
	}, nil
}

func (Documents) GetHighlights(_ context.Context, _ []byte, _ []int, pages []int) ([]domain.HighlightBox, error) {
	var boxes []domain.HighlightBox
	for _, p := range pages {
		if p == 1 {
			boxes = append(boxes, domain.HighlightBox{
				Page:     1,
				ByteSpan: domain.ByteSpan{From: 17, To: 22},
				Rect:     domain.Rect{MinX: 0.1, MinY: 0.1, MaxX: 0.5, MaxY: 0.2},
			})
		}
	}
	return boxes, nil
}

type dummyOpenDocument struct {
	file io.Closer
}

func (d dummyOpenDocument) GetPageCount() int {
	return 1
}

func (d dummyOpenDocument) GetPageSize(_ int) (float64, float64, error) {
	return 100, 100, nil
}

func (d dummyOpenDocument) RenderPage(_, _ int) (image.Image, error) {
	return image.NewRGBA(image.Rect(0, 0, 100, 100)), nil
}

func (d dummyOpenDocument) Close() {
	if d.file != nil {
		_ = d.file.Close()
	}
}

func (Documents) OpenDocument(_ context.Context, file io.ReadSeekCloser) (port.OpenDocument, error) {
	if file != nil {
		buf := make([]byte, 5)
		n, _ := io.ReadFull(file, buf)
		if n < 5 || !bytes.HasPrefix(buf, []byte("%PDF-")) {
			_ = file.Close()
			return nil, port.ErrNotADocument
		}
		_, _ = file.Seek(0, io.SeekStart)
	}
	return dummyOpenDocument{file: file}, nil
}

// bookMark opens the bytes of a book the stub reads. The words follow it, and
// each line of them is a part.
const bookMark = "%BOOK\n"

// NewBook is the bytes of a book the stub reads: the words, one line to a part.
func NewBook(words ...string) []byte {
	return []byte(bookMark + strings.Join(words, "\n"))
}

// ReadBook reads the bytes NewBook gives: the lines are the text, and a part
// begins at each line.
func (Documents) ReadBook(_ context.Context, raw []byte) (domain.BookText, error) {
	if !bytes.HasPrefix(raw, []byte(bookMark)) {
		return domain.BookText{}, port.ErrNotADocument
	}
	text := string(raw[len(bookMark):])
	out := domain.BookText{Text: text}
	offset := 0
	for i, line := range strings.Split(text, "\n") {
		out.Parts = append(out.Parts, domain.PartStart{Title: "Part " + strconv.Itoa(i+1), Offset: offset})
		offset += len(line) + 1
	}
	return out, nil
}
