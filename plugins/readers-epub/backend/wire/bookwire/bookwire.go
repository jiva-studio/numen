// Package bookwire answers the book methods of the md.numen.readers.epub service: it turns their
// requests into calls of the books use case, and its books and failures into responses and API errors.
package bookwire

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
)

// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mostListed caps the maximum number of items returned in a book summary"
const mostListed = 1000

// Methods answers the book methods with a books.Reader.
type Methods struct {
	reader *books.Reader
}

// NewMethods returns the book methods that reader answers.
func NewMethods(reader *books.Reader) *Methods {
	return &Methods{reader: reader}
}

// GetBook answers the metadata, documents and parts of a book.
func (m *Methods) GetBook(ctx context.Context, req api.GetBookRequest) (api.GetBookResponse, error) {
	book, err := m.reader.GetBook(ctx, req.Path)
	if err != nil {
		return api.GetBookResponse{}, ToAPIError(err)
	}
	return ToGetBookResponse(book), nil
}

// ReadBookMarkup answers the rendered HTML of a document of a book's spine.
func (m *Methods) ReadBookMarkup(ctx context.Context, req api.ReadBookMarkupRequest) (api.ReadBookMarkupResponse, error) {
	markup, err := m.reader.ReadBookMarkup(ctx, req.Path, req.Document)
	if err != nil {
		return api.ReadBookMarkupResponse{}, ToAPIError(err)
	}
	return api.ReadBookMarkupResponse{Markup: markup}, nil
}

// ReadBookEntry answers the bytes of one entry of a book's archive.
func (m *Methods) ReadBookEntry(ctx context.Context, req api.ReadBookEntryRequest) (api.ReadBookEntryResponse, error) {
	body, err := m.reader.ReadBookEntry(ctx, req.Path, req.Entry, ToBaseline(req.Baseline))
	if err != nil {
		return api.ReadBookEntryResponse{}, ToAPIError(err)
	}
	return api.ReadBookEntryResponse{Body: body}, nil
}

// GetCfi answers the EPUB CFI of the character at an offset of a book's text.
func (m *Methods) GetCfi(ctx context.Context, req api.GetCfiRequest) (api.GetCfiResponse, error) {
	place, err := m.reader.GetCfi(ctx, req.Path, req.Offset)
	if err != nil {
		return api.GetCfiResponse{}, ToAPIError(err)
	}
	return api.GetCfiResponse{Cfi: place}, nil
}

// FindOffset answers the offset of a book's text an EPUB CFI names.
func (m *Methods) FindOffset(ctx context.Context, req api.FindOffsetRequest) (api.FindOffsetResponse, error) {
	offset, err := m.reader.FindOffset(ctx, req.Path, req.Cfi)
	if err != nil {
		return api.FindOffsetResponse{}, ToAPIError(err)
	}
	return api.FindOffsetResponse{Offset: offset}, nil
}

// ToGetBookResponse returns the metadata of book and at most mostListed of its documents, parts and
// printed pages each.
func ToGetBookResponse(book *epub.Book) api.GetBookResponse {
	res := api.GetBookResponse{
		Title:        book.Title,
		PageCount:    book.PageCount(),
		TextBytes:    len(book.Text),
		PageBytes:    book.PageBytes(),
		Documents:    []api.SpineDocument{},
		Parts:        []api.BookPart{},
		PrintedPages: []api.PrintedPage{},
		Stat:         nil,
	}

	docCount := min(len(book.Documents), mostListed)
	for _, doc := range book.Documents[:docCount] {
		res.Documents = append(res.Documents, api.SpineDocument{Path: doc.Path, Offset: doc.Offset, Length: doc.Length})
	}

	partCount := min(len(book.Parts), mostListed)
	for _, part := range book.Parts[:partCount] {
		res.Parts = append(res.Parts, api.BookPart{Title: part.Title, Offset: part.Offset, Level: part.Level})
	}

	pageCount := min(len(book.Pages), mostListed)
	for _, page := range book.Pages[:pageCount] {
		res.PrintedPages = append(res.PrintedPages, api.PrintedPage{Label: page.Label, Offset: page.Offset})
	}

	return res
}

// ToBaseline returns the size and time of stat, and nil for no stat.
func ToBaseline(stat *api.FileStat) *books.Baseline {
	if stat == nil {
		return nil
	}
	return &books.Baseline{Size: stat.Size, Mtime: stat.Mtime}
}

// ToAPIError returns the API error of a failure of the books use case, with the failure's text as its
// message. A failure of another kind is returned as it is.
func ToAPIError(err error) error {
	failure, ok := errors.AsType[*books.Error](err)
	if !ok {
		return err
	}
	return server.NewAPIError(getCode(failure.Kind), failure.Message)
}

func getCode(kind books.Kind) server.ErrorCode {
	switch kind {
	case books.KindInvalidArgument:
		return server.CodeInvalidArgument
	case books.KindNotFound:
		return server.CodeNotFound
	case books.KindPermissionDenied:
		return server.CodePermissionDenied
	case books.KindFailedPrecondition:
		return server.CodeFailedPrecondition
	case books.KindUnavailable:
		return server.CodeUnavailable
	}
	return server.CodeInternal
}
