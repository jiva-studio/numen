package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/api"
)

func postDocument(t *testing.T, baseURL, token, body string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/"+api.Service+"/GetDocument", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	return res.StatusCode
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

// The mounted plugin answers the document API with the token it was given and rejects a call without it.
func TestTheMountedPluginAnswersTheDocumentAPI(t *testing.T) {
	handler := newMountedHandler(t, "1")
	web := httptest.NewServer(handler)
	defer web.Close()

	if status := postDocument(t, web.URL, "", `{"path":"tiny.pdf"}`); status != http.StatusUnauthorized {
		t.Fatalf("GetDocument without the token = %d, want 401", status)
	}
	if status := postDocument(t, web.URL, "linked-token", `{}`); status != http.StatusBadRequest {
		t.Fatalf("GetDocument with no path = %d, want 400", status)
	}
}

// The mounted plugin connects to the host at the port set in the environment.
func TestTheMountedPluginReachesTheHostOfItsEnvironment(t *testing.T) {
	hostServer := httptest.NewServer(http.NotFoundHandler())
	defer hostServer.Close()
	port := hostServer.URL[strings.LastIndex(hostServer.URL, ":")+1:]

	handler := newMountedHandler(t, port)
	web := httptest.NewServer(handler)
	defer web.Close()

	status := postDocument(t, web.URL, "linked-token", `{"path":"books/tiny.pdf"}`)

	if status != http.StatusNotFound {
		t.Fatalf("GetDocument = %d, want 404", status)
	}
}
