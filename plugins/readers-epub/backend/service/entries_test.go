package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
)

// serveEntries is the entries route over a book server whose current vault holds nothing, beside the
// vault Chemistry holding the picture book.
func serveEntries(t *testing.T) (*http.ServeMux, *BookServer) {
	t.Helper()
	current := hosttest.OnDisk(t, "Physics", map[string]string{"Note.md": "# Note\n"})
	other := hosttest.OnDisk(t, "Chemistry", map[string]string{"pictured.epub": string(createPictureEpub(t))})
	current.Beside(other)
	books := NewBookServer(current, func() string { return current.Vault().Path })
	t.Cleanup(books.Close)
	mux := http.NewServeMux()
	mux.Handle("GET /"+entriesPattern, NewEntryServer(books))
	return mux, books
}

func TestTheEntriesRouteServesAPictureOfTheBookInTheVaultTheRequestNames(t *testing.T) {
	mux, _ := serveEntries(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub", nil)
	req.Header.Set("Numen-Vault", "Chemistry")
	answer := httptest.NewRecorder()
	mux.ServeHTTP(answer, req)

	if answer.Code != http.StatusOK || !bytes.Equal(answer.Body.Bytes(), pngBytes) {
		t.Fatalf("the picture answered %d with %d bytes", answer.Code, answer.Body.Len())
	}
	current := httptest.NewRecorder()
	mux.ServeHTTP(current, httptest.NewRequest(http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub", nil))
	if current.Code == http.StatusOK {
		t.Error("a request naming no vault served a book the current vault does not hold")
	}
}

func TestTheEntriesRouteRefusesABookThatChanged(t *testing.T) {
	mux, _ := serveEntries(t)

	req := httptest.NewRequest(http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub&size=1&mtime=1", nil)
	req.Header.Set("Numen-Vault", "Chemistry")
	answer := httptest.NewRecorder()
	mux.ServeHTTP(answer, req)

	if answer.Code != http.StatusConflict {
		t.Errorf("a stale picture answered %d, want %d", answer.Code, http.StatusConflict)
	}
}

func TestGetBookReturnsTheCFIOfEachBlock(t *testing.T) {
	dir := t.TempDir()
	raw := createPictureEpub(t)
	if err := os.WriteFile(filepath.Join(dir, "pictured.epub"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	books := NewBookServer(hosttest.New("vault"), func() string { return dir })
	t.Cleanup(books.Close)

	res, err := books.GetBook(t.Context(), connect.NewRequest(&epubv1.GetBookRequest{Path: "pictured.epub"}))
	if err != nil {
		t.Fatal(err)
	}

	cfis := map[string]bool{}
	for _, anchor := range res.Msg.GetAnchors() {
		cfis[anchor.GetCfi()] = true
	}
	if !cfis["epubcfi(/6/2!/4/2)"] || !cfis["epubcfi(/6/4!/4/4)"] {
		t.Errorf("anchors %v, want the heading of the first chapter and the paragraph of the second", res.Msg.GetAnchors())
	}
}
