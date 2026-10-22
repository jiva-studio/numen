package document

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/jpeg"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// ErrNoPage is a page the document does not have.
var ErrNoPage = errors.New("no such page in this document")

// quality is what a drawn page is encoded at. A page of a scan is a photograph
// of paper, which JPEG carries, and at this quality it loses less than the
// grain of the paper.
const quality = 82

// GetPage draws one page of a document as a JPEG.
type GetPage struct {
	readers   vaultport.VaultReaders
	documents Documents
	pages     Pages
}

func NewGetPage(readers vaultport.VaultReaders, documents Documents, pages Pages) GetPage {
	return GetPage{readers: readers, documents: documents, pages: pages}
}

// Execute answers while the file at the path is the bytes the key names by size
// and stamp, and refuses it with ErrChanged otherwise. Several asks for one page
// draw it once.
func (u GetPage) Execute(ctx context.Context, v domain.Vault, requested PageKey) ([]byte, error) {
	reader, mark, err := getFingerprint(ctx, u.readers, v, requested.Document.Path)
	if err != nil {
		return nil, err
	}
	if err := checkUnchanged(requested.Document, mark); err != nil {
		return nil, err
	}
	key := PageKey{Document: mark, Page: requested.Page, Width: requested.Width}
	return u.pages.GetPage(ctx, key, func() ([]byte, error) {
		return u.draw(ctx, reader, key)
	})
}

// draw is one page of a document, drawn and encoded.
func (u GetPage) draw(ctx context.Context, reader vaultport.VaultReader, key PageKey) ([]byte, error) {
	var body []byte
	err := u.documents.UseDocument(ctx, key.Document, openFile(reader, key.Document.Path), func(scan Scan) error {
		if key.Page >= scan.GetPageCount() {
			return fmt.Errorf("%w: page %d of %d", ErrNoPage, key.Page, scan.GetPageCount())
		}
		rendered, err := scan.Render(key.Page, key.Width)
		if err != nil {
			return err
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, rendered, &jpeg.Options{Quality: quality}); err != nil {
			return err
		}
		body = out.Bytes()
		return nil
	})
	return body, err
}
