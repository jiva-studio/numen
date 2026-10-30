package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

// serveEntries is the entries route over a book server whose current vault is
// empty, plus the vault Chemistry that contains the image book.
func serveEntries(t *testing.T) (*http.ServeMux, *BookServer) {
	t.Helper()
	current := hosttest.OnDisk(t, "Physics", map[string]string{"Note.md": "# Note\n"})
	other := hosttest.OnDisk(t, "Chemistry", map[string]string{"pictured.epub": string(createImageEpub(t))})
	current.Beside(other)
	books := NewBookServer(current)
	t.Cleanup(books.Close)
	mux := http.NewServeMux()
	mux.Handle("GET /"+entriesPattern, NewEntryServer(books))
	return mux, books
}

func TestEntriesRoute_ServesBookImageFromRequestedVault(t *testing.T) {
	mux, _ := serveEntries(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub", nil)
	req.Header.Set("Numen-Vault", "Chemistry")
	answer := httptest.NewRecorder()
	mux.ServeHTTP(answer, req)

	if answer.Code != http.StatusOK || !bytes.Equal(answer.Body.Bytes(), pngBytes) {
		t.Fatalf("the picture answered %d with %d bytes", answer.Code, answer.Body.Len())
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
