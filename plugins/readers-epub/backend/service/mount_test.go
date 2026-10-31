package service

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// codeOf returns the code a method failing with err answers.
func codeOf(err error) server.ErrorCode {
	if err == nil {
		return ""
	}
	return toAPIError(err).Code
}

func postMethod(t *testing.T, handler http.Handler, method, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/numen.epub/"+method, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorded := httptest.NewRecorder()
	handler.ServeHTTP(recorded, req)
	return recorded
}

func TestLinkedHandler_RequiresTheHostToken(t *testing.T) {
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	if got := postMethod(t, handler, "GetBook", `{"path":"book.epub"}`, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("GetBook without the token = %d, want %d", got, http.StatusUnauthorized)
	}
	if got := postMethod(t, handler, "GetBook", `{}`, "linked-token"); got.Code != http.StatusBadRequest || !strings.Contains(got.Body.String(), `"code":"invalid_argument"`) {
		t.Fatalf("GetBook with no path = %d %s, want invalid_argument", got.Code, got.Body)
	}
}

func TestLinkedHandler_ReachesTheHostOfItsVariables(t *testing.T) {
	hostServer := httptest.NewServer(http.NotFoundHandler())
	defer hostServer.Close()
	port := hostServer.URL[strings.LastIndex(hostServer.URL, ":")+1:]

	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": port})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	body := `{"path":"` + filepath.Join("books", "book.epub") + `"}`
	if got := postMethod(t, handler, "GetBook", body, "linked-token"); got.Code != http.StatusNotFound {
		t.Fatalf("GetBook = %d %s, want %d", got.Code, got.Body, http.StatusNotFound)
	}
}
