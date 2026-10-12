package document

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetBook reads the book at a path.
type GetBook struct {
	readers port.VaultReaders
	books   Books
}

func NewGetBook(readers port.VaultReaders, books Books) GetBook {
	return GetBook{readers: readers, books: books}
}

func (u GetBook) Execute(ctx context.Context, v domain.Vault, path string) (Document, error) {
	read, mark, err := readBook(ctx, u.readers, u.books, v, path, nil)
	if err != nil {
		return Document{}, err
	}
	return Document{Fingerprint: mark, Book: read}, nil
}

// readBook is the book at a path, held where it is read already and refused with
// ErrChanged where the file is not the bytes the baseline names.
func readBook(
	ctx context.Context, readers port.VaultReaders, books Books,
	v domain.Vault, path string, baseline *Fingerprint,
) (*epub.Book, Fingerprint, error) {
	reader, mark, err := getFingerprint(ctx, readers, v, path)
	if err != nil {
		return nil, Fingerprint{}, err
	}
	if baseline != nil {
		if err := checkUnchanged(*baseline, mark); err != nil {
			return nil, Fingerprint{}, err
		}
	}
	read, err := books.GetBook(ctx, mark, func() (*epub.Book, error) {
		raw, err := reader.Read(context.WithoutCancel(ctx), mark.Path)
		if err != nil {
			return nil, err
		}
		return epub.Read(raw)
	})
	if err != nil {
		return nil, Fingerprint{}, err
	}
	return read, mark, nil
}
