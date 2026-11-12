package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// httpResponse is the status and the headers of a response whose body was read and closed.
type httpResponse struct {
	StatusCode int
	Header     http.Header
}

func sendRequest(t *testing.T, method, target string, headers map[string]string) httpResponse {
	t.Helper()
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":%s}`, initializeParams))
	}
	req, err := http.NewRequestWithContext(t.Context(), method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, res.Body); err != nil {
		t.Error(err)
	}
	if err := res.Body.Close(); err != nil {
		t.Error(err)
	}
	return httpResponse{StatusCode: res.StatusCode, Header: res.Header}
}

func assertUnauthorizedResponse(t *testing.T, res httpResponse, what string) {
	t.Helper()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("%s returned %d, want 401", what, res.StatusCode)
	}
	if got := res.Header.Get("WWW-Authenticate"); got != `Bearer realm="numen"` {
		t.Errorf("%s returned WWW-Authenticate %q, want Bearer realm=\"numen\"", what, got)
	}
}
