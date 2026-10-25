package document

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Document is what GetDocument reads: which bytes it came from, and the size of
// each page, wide and high in the page's own units.
type Document struct {
	Fingerprint Fingerprint
	Pages       [][2]float64
}

// GetDocument finds how many pages a document has and how big each of them is.
type GetDocument struct {
	readers   vaultport.VaultReaders
	documents Documents
}

func NewGetDocument(readers vaultport.VaultReaders, documents Documents) GetDocument {
	return GetDocument{readers: readers, documents: documents}
}

// Execute answers a page whose size could not be read with the size zero, and
// the pages after it with theirs.
func (u GetDocument) Execute(ctx context.Context, v domain.Vault, path string) (Document, error) {
	reader, mark, err := getFingerprint(ctx, u.readers, v, path)
	if err != nil {
		return Document{}, err
	}
	out := Document{Fingerprint: mark}
	err = u.documents.UseDocument(ctx, mark, openFile(reader, mark.Path), func(scan Scan) error {
		out.Pages = make([][2]float64, scan.GetPageCount())
		for i := range out.Pages {
			if width, height, err := scan.GetPageSize(i); err == nil {
				out.Pages[i] = [2]float64{width, height}
			}
		}
		return nil
	})
	if err != nil {
		return Document{}, err
	}
	return out, nil
}

// openFile is the file of a document, opened as it is drawn from.
func openFile(reader vaultport.VaultReader, path string) func(context.Context) (io.ReadSeekCloser, error) {
	return func(ctx context.Context) (io.ReadSeekCloser, error) {
		return reader.OpenShared(ctx, path)
	}
}
