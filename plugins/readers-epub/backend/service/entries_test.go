package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/books"
)

func serveEntries(t *testing.T) (*http.ServeMux, *books.Reader) {
	t.Helper()
	current := hosttest.OnDisk(t, "Physics", map[string]string{"Note.md": "# Note\n"})
	other := hosttest.OnDisk(t, "Chemistry", map[string]string{"pictured.epub": string(testsupport.CreateImageEpub(t))})
	current.Beside(other)
	reader := books.NewReader(func(id domain.VaultID) host.FileReader { return current.With(id) })
	t.Cleanup(reader.Close)
	mux := http.NewServeMux()
	mux.Handle("GET /"+entriesPattern, NewEntryServer(reader, func(string, error) {}))
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
	badMtime := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/entries/EPUB/cover.png?path=pictured.epub&size="+strconv.FormatInt(int64(len(testsupport.CreateImageEpub(t))), 10)+"&mtime=notanumber", nil)
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
