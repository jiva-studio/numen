package document

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetBookMarkup reads one document of a book as the markup a page is set from.
type GetBookMarkup struct {
	readers port.VaultReaders
	books   Books
}

func NewGetBookMarkup(readers port.VaultReaders, books Books) GetBookMarkup {
	return GetBookMarkup{readers: readers, books: books}
}

// Execute answers while the file at the path is the bytes the baseline names,
// where there is one, and refuses it with ErrChanged otherwise.
func (u GetBookMarkup) Execute(
	ctx context.Context, v domain.Vault, path string, baseline *Fingerprint, document string,
) (string, error) {
	read, _, err := readBook(ctx, u.readers, u.books, v, path, baseline)
	if err != nil {
		return "", err
	}
	drawn, err := read.Markup(document)
	if err != nil {
		return "", err
	}
	return drawn.HTML(), nil
}
