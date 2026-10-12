package editor

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// Documents is what a window reads a document of the vault with: its pages and
// its books.
type Documents struct {
	GetDocument    *document.GetDocument
	GetPage        *document.GetPage
	GetBook        *document.GetBook
	GetBookMarkup  *document.GetBookMarkup
	GetBookPicture *document.GetBookPicture
}

// newDocuments reads documents of the vaults the readers open, through what the
// viewer holds open.
func newDocuments(readers port.VaultReaders, v *viewer) Documents {
	getDocument := document.NewGetDocument(readers, v)
	getPage := document.NewGetPage(readers, v, v)
	getBook := document.NewGetBook(readers, v)
	getBookMarkup := document.NewGetBookMarkup(readers, v)
	getBookPicture := document.NewGetBookPicture(readers, v)
	return Documents{
		GetDocument:    &getDocument,
		GetPage:        &getPage,
		GetBook:        &getBook,
		GetBookMarkup:  &getBookMarkup,
		GetBookPicture: &getBookPicture,
	}
}
