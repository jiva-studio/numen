package extract

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/textlayer"
)

// layer is one file's bytes and the document read from them.
type layer struct {
	hash string
	raw  []byte
	book *pdf.Book
}

func (l layer) hasText() bool { return strings.TrimSpace(l.book.Text) != "" }

func (l layer) getSegments(workers *pdf.Workers) ([]domain.TextSegment, error) {
	return textlayer.GetSegments(l.book, func(pages []int) ([]pdf.HighlightBox, error) {
		return l.book.Highlights(workers, l.raw, pages)
	})
}
