package testsupport

import (
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Box is one run of prose on a page: at its current position in the text, and the
// fraction of the page it covers.
func Box(page, from, to int, finished domain.Rect) domain.HighlightBox {
	return domain.HighlightBox{
		Page:     page,
		ByteSpan: domain.ByteSpan{From: from, To: to},
		Rect:     finished,
	}
}
