package bookwire

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
)

func newMethods(t *testing.T) (*Methods, *books.Reader) {
	t.Helper()
	files := hosttest.New("vault")
	files.File("guided.epub", string(testsupport.CreateGuidedEpub(t)))
	files.File("pictured.epub", string(testsupport.CreateImageEpub(t)))
	files.File("enc.epub", string(testsupport.CreateEncryptedEpub(t)))
	files.File("bad.epub", "not an epub")
	reader := books.NewReader(func(id domain.VaultID) host.FileReader { return files.With(id) })
	t.Cleanup(reader.Close)
	return NewMethods(reader), reader
}

func checkAPIError(t *testing.T, err error, code server.ErrorCode, message string) {
	t.Helper()
	failure, ok := errors.AsType[*server.APIError](err)
	if !ok {
		t.Fatalf("error %v is no API error", err)
	}
	if failure.Code != code || !strings.HasPrefix(failure.Message, message) {
		t.Errorf("error is %s %q, want %s %q", failure.Code, failure.Message, code, message)
	}
}

func TestMethodsAnswerTheCodeAndMessageOfEachFailure(t *testing.T) {
	methods, _ := newMethods(t)
	ctx := t.Context()

	_, err := methods.GetBook(ctx, api.GetBookRequest{Path: ""})
	checkAPIError(t, err, server.CodeInvalidArgument, "path required")

	_, err = methods.GetBook(ctx, api.GetBookRequest{Path: "missing.epub"})
	checkAPIError(t, err, server.CodeNotFound, "read book missing.epub: ")

	_, err = methods.GetBook(ctx, api.GetBookRequest{Path: "../outside.epub"})
	checkAPIError(t, err, server.CodePermissionDenied, "path traversal outside allowed scope: path outside allowed vault root")

	_, err = methods.GetBook(ctx, api.GetBookRequest{Path: "bad.epub"})
	checkAPIError(t, err, server.CodeInvalidArgument, "parse epub bad.epub: ")

	_, err = methods.ReadBookEntry(ctx, api.ReadBookEntryRequest{Path: "pictured.epub", Entry: ""})
	checkAPIError(t, err, server.CodeInvalidArgument, "entry required")

	_, err = methods.ReadBookEntry(ctx, api.ReadBookEntryRequest{Path: "enc.epub", Entry: "EPUB/c1.xhtml"})
	checkAPIError(t, err, server.CodeFailedPrecondition, "read epub: ")

	_, err = methods.ReadBookEntry(ctx, api.ReadBookEntryRequest{Path: "pictured.epub", Entry: "EPUB/cover.png", Baseline: &api.FileStat{Path: "", Size: 1, Mtime: 0}})
	checkAPIError(t, err, server.CodeFailedPrecondition, "read book pictured.epub: the file changed")

	_, err = methods.FindOffset(ctx, api.FindOffsetRequest{Path: "guided.epub", Cfi: "epubcfi(/6/4!/4/2/1:0)"})
	checkAPIError(t, err, server.CodeNotFound, "epubcfi(/6/4!/4/2/1:0) names no document of guided.epub")
}

func TestReadBookEntryAfterCloseIsUnavailable(t *testing.T) {
	methods, reader := newMethods(t)
	reader.Close()

	_, err := methods.ReadBookEntry(t.Context(), api.ReadBookEntryRequest{Path: "pictured.epub", Entry: "EPUB/cover.png"})

	checkAPIError(t, err, server.CodeUnavailable, "read book pictured.epub: the book cache is closed")
}

func TestToAPIErrorReturnsAnotherFailureAsItIs(t *testing.T) {
	failure := errors.New("disk on fire")

	if got := ToAPIError(failure); got != failure {
		t.Errorf("ToAPIError = %v, want the failure itself", got)
	}
}

func TestMethodsAnswerTheBookAndItsPlaces(t *testing.T) {
	methods, _ := newMethods(t)
	ctx := t.Context()

	book, err := methods.GetBook(ctx, api.GetBookRequest{Path: "guided.epub"})
	if err != nil || len(book.Documents) < 2 {
		t.Fatalf("GetBook = %+v (%v), want two documents", book, err)
	}
	place, err := methods.GetCfi(ctx, api.GetCfiRequest{Path: "guided.epub", Offset: book.Documents[1].Offset + len("Epilogue of the ")})
	if err != nil || place.Cfi != "epubcfi(/8/4!/4/2/1:17)" {
		t.Errorf("GetCfi = %+v (%v), want epubcfi(/8/4!/4/2/1:17)", place, err)
	}
	offset, err := methods.FindOffset(ctx, api.FindOffsetRequest{Path: "guided.epub", Cfi: "epubcfi(/8/4!/4/2/1:17)"})
	if err != nil || offset.Offset != book.Documents[1].Offset+len("Epilogue of the ") {
		t.Errorf("FindOffset = %+v (%v)", offset, err)
	}
}
