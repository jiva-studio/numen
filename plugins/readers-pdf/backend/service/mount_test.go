package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postDocument(t *testing.T, baseURL, token, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL+"/"+APIService+"/GetDocument", strings.NewReader(body))
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
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

// The handler a host that links the plugin in answers the document API with the token it was given
// and rejects a call without it.
func TestTheLinkedHandlerAnswersTheDocumentAPI(t *testing.T) {
	handler, err := NewHandler(map[string]string{
		"NUMEN_PLUGIN_TOKEN": "linked-token",
		"NUMEN_HOST_PORT":    "1",
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	if res := postDocument(t, web.URL, "", `{"path":"tiny.pdf"}`); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GetDocument without the token = %d, want 401", res.StatusCode)
	}
	if res := postDocument(t, web.URL, "linked-token", `{}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("GetDocument with no path = %d, want 400", res.StatusCode)
	}
}

// A linked handler reaches the host at the port of its variables.
func TestTheLinkedHandlerReachesTheHostOfItsVariables(t *testing.T) {
	hostServer := httptest.NewServer(http.NotFoundHandler())
	defer hostServer.Close()
	port := hostServer.URL[strings.LastIndex(hostServer.URL, ":")+1:]

	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "linked-token", "NUMEN_HOST_PORT": port})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	web := httptest.NewServer(handler)
	defer web.Close()

	res := postDocument(t, web.URL, "linked-token", `{"path":"books/tiny.pdf"}`)

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GetDocument = %d, want 404", res.StatusCode)
	}
}
