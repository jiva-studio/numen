// Package document reads the PDF files of a vault: the size of their pages, the places of runs of
// their text and the pages drawn as images.
package document

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// ErrOpen is the failure to open a file as a PDF document.
var ErrOpen = errors.New("read pdf")

// ErrPageSize is the failure to measure a page.
var ErrPageSize = errors.New("page size")

// ErrClose is the failure to close a document after it was read.
var ErrClose = errors.New("close pdf")

// Reader reads files from the vault a call names, on the workers, and text versions from the same
// vault.
type Reader struct {
	files   func(domain.VaultID) host.FileReader
	texts   func(domain.VaultID) LiveTexts
	workers *pdf.Workers
}

// NewReader returns a Reader that reads files from vaults on the workers, and text versions from
// texts.
func NewReader(files func(domain.VaultID) host.FileReader, texts func(domain.VaultID) LiveTexts, workers *pdf.Workers) *Reader {
	return &Reader{files: files, texts: texts, workers: workers}
}

// Page is the size of one page.
type Page struct {
	Width  float64
	Height float64
}

// GetPages returns the size of each page of the PDF file in the vault, in order.
func (r *Reader) GetPages(ctx context.Context, vault domain.VaultID, path string) (pages []Page, err error) {
	raw, _, err := r.files(vault).Read(ctx, path)
	if err != nil {
		return nil, err
	}
	scan, err := r.workers.OpenBytes(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOpen, err)
	}
	defer func() {
		if closeErr := scan.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("%w: %w", ErrClose, closeErr))
		}
	}()

	pageCount := scan.GetPageCount()
	pages = make([]Page, 0, pageCount)
	for i := range pageCount {
		w, h, err := scan.GetPageSize(i)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrPageSize, err)
		}
		pages = append(pages, Page{Width: w, Height: h})
	}
	return pages, nil
}
