package service

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/wire/bookwire"
)

type failingResponseWriter struct {
	http.ResponseWriter
	err error
}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func serveEntries(t *testing.T) (*http.ServeMux, *books.Reader) {
	t.Helper()
	current := hosttest.CreateOnDisk(t, "Physics", map[string]string{"Note.md": "# Note\n"})
	other := hosttest.CreateOnDisk(t, "Chemistry", map[string]string{"pictured.epub": string(testsupport.CreateImageEpub(t))})
	current.LinkBeside(other)
	reader := books.NewReader(func(id domain.VaultID) host.FileReader { return current.With(id) })
	t.Cleanup(reader.Close)
	mux := http.NewServeMux()
	mux.Handle("GET /"+entriesPattern, NewEntryServer(bookwire.NewMethods(reader), func(string, error) {}))
	return mux, reader
}

func TestEntriesRoute_ServesBookImageFromRequestedVault(t *testing.T) {
	mux, _ := serveEntries(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub", nil)
	req.Header.Set("Numen-Vault", "Chemistry")
	answer := httptest.NewRecorder()
	mux.ServeHTTP(answer, req)

	if answer.Code != http.StatusOK || !bytes.Equal(answer.Body.Bytes(), testsupport.GetPNGBytes()) {
		t.Fatalf("the picture answered %d with %d bytes", answer.Code, answer.Body.Len())
	}
	if got := answer.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want immutable", got)
	}
	badMtime := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/entries/EPUB/cover.png?path=pictured.epub&size="+strconv.FormatInt(int64(len(testsupport.CreateImageEpub(t))), 10)+"&mtime=notanumber",
		nil,
	)
	badMtime.Header.Set("Numen-Vault", "Chemistry")
	badMtimeAnswer := httptest.NewRecorder()
	mux.ServeHTTP(badMtimeAnswer, badMtime)
	if badMtimeAnswer.Code != http.StatusOK {
		t.Errorf("invalid mtime query answered %d, want 200", badMtimeAnswer.Code)
	}
	current := httptest.NewRecorder()
	mux.ServeHTTP(current, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub", nil))
	if current.Code == http.StatusOK {
		t.Error("a request naming no vault served a book the current vault does not hold")
	}
}

func TestEntriesRoute_ConflictOnChangedBook(t *testing.T) {
	mux, _ := serveEntries(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub&size=1&mtime=1", nil)
	req.Header.Set("Numen-Vault", "Chemistry")
	answer := httptest.NewRecorder()
	mux.ServeHTTP(answer, req)

	if answer.Code != http.StatusConflict {
		t.Errorf("a stale picture answered %d, want %d", answer.Code, http.StatusConflict)
	}
}

func TestEntriesRoute_ReportsWriteError(t *testing.T) {
	_, reader := serveEntries(t)
	var gotWhat string
	var gotErr error
	mux := http.NewServeMux()
	mux.Handle("GET /"+entriesPattern, NewEntryServer(bookwire.NewMethods(reader), func(what string, err error) {
		gotWhat = what
		gotErr = err
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub", nil)
	req.Header.Set("Numen-Vault", "Chemistry")
	writeErr := errors.New("broken pipe")
	mux.ServeHTTP(&failingResponseWriter{ResponseWriter: httptest.NewRecorder(), err: writeErr}, req)

	if gotWhat != "write the entry EPUB/cover.png" || !errors.Is(gotErr, writeErr) {
		t.Errorf("reported (%q, %v), want (\"write the entry EPUB/cover.png\", %v)", gotWhat, gotErr, writeErr)
	}
}
