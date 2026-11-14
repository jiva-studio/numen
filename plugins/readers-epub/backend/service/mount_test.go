package service

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

func postMethod(t *testing.T, handler http.Handler, method, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/md.numen.readers.epub/"+method, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorded := httptest.NewRecorder()
	handler.ServeHTTP(recorded, req)
	return recorded
}

func newMountedHandler(t *testing.T, port string) http.Handler {
	t.Helper()
	t.Setenv(hostcall.HostPortEnv, port)
	t.Setenv(hostcall.PluginTokenEnv, "linked-token")
	handler, err := server.NewLinkedHandler(map[string]string{hostcall.PluginTokenEnv: "linked-token"}, Mount)
	if err != nil {
		t.Fatalf("NewLinkedHandler: %v", err)
	}
	return handler
}

func TestMountedHandler_RequiresTheHostToken(t *testing.T) {
	handler := newMountedHandler(t, "1")

	if got := postMethod(t, handler, "GetBook", `{"path":"book.epub"}`, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("GetBook without the token = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := postMethod(t, handler, "GetBook", `{}`, "linked-token"); got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), `"code":"invalid_argument"`) {
		t.Fatalf("GetBook with no path = %d %s, want invalid_argument", got.Code, got.Body)
	}
	for _, method := range []string{"ReadBookMarkup", "ReadBookEntry", "GetCfi", "FindOffset"} {
		if got := postMethod(t, handler, method, `{}`, "linked-token"); got.Code != http.StatusBadRequest {
			t.Errorf("%s with empty body = %d, want 400", method, got.Code)
		}
	}
	entryReq := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/plugin/entries/EPUB/cover.png", nil)
	entryReq.Header.Set("Authorization", "Bearer linked-token")
	entryRec := httptest.NewRecorder()
	handler.ServeHTTP(entryRec, entryReq)
	if entryRec.Code != http.StatusBadRequest {
		t.Errorf("GET /plugin/entries without path = %d, want 400", entryRec.Code)
	}
}

func TestMountedHandler_ReachesTheHostOfItsEnvironment(t *testing.T) {
	hostServer := httptest.NewServer(http.NotFoundHandler())
	defer hostServer.Close()
	port := hostServer.URL[strings.LastIndex(hostServer.URL, ":")+1:]

	handler := newMountedHandler(t, port)

	body := `{"path":"` + filepath.Join("books", "book.epub") + `"}`
	if got := postMethod(t, handler, "GetBook", body, "linked-token"); got.Code != http.StatusNotFound {
		t.Fatalf("GetBook = %d %s, want %d", got.Code, got.Body, http.StatusNotFound)
	}
}
