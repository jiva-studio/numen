package mcphttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const gardenToken = "garden-token-4417"

const initializeParams = `{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"orchard","version":"1"}}`

func newServedHandler(t *testing.T, token string) (*Server, string) {
	t.Helper()
	srv := newTestServer(token)
	served := httptest.NewServer(srv.Handler())
	t.Cleanup(served.Close)
	return srv, served.URL
}

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

func TestARequestWithTheBearerTokenIsServed(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Authorization": "Bearer " + gardenToken})

	if res.StatusCode != http.StatusOK {
		t.Errorf("an initialize with the token returned %d, want 200", res.StatusCode)
	}
}

func TestARequestWithAGeneratedHexTokenIsServed(t *testing.T) {
	t.Parallel()
	const generated = "a1b2c3d4e5f60718a1b2c3d4e5f60718a1b2c3d4e5f60718a1b2c3d4e5f60718"
	_, url := newServedHandler(t, generated)

	res := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Authorization": "Bearer " + generated})

	if res.StatusCode != http.StatusOK {
		t.Errorf("an initialize with a 64-character hex token returned %d, want 200", res.StatusCode)
	}
	assertUnauthorizedResponse(t, sendRequest(t, http.MethodPost, url+"/mcp",
		map[string]string{"Authorization": "Bearer " + generated[:63]}), "an initialize with the hex token cut short")
}

func TestARequestWithoutTheTokenGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp", nil)

	assertUnauthorizedResponse(t, res, "an initialize without the token")
}

func TestARequestWithAWrongTokenGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Authorization": "Bearer garden-token-4418"})

	assertUnauthorizedResponse(t, res, "an initialize with a wrong token")
}

func TestATokenInTheQueryGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp?token="+gardenToken, nil)

	assertUnauthorizedResponse(t, res, "an initialize with the token in the query")
}

func TestAServerMadeWithoutATokenHasAnEmptyToken(t *testing.T) {
	t.Parallel()

	srv := newTestServer("")

	if got := srv.GetToken(); got != "" {
		t.Errorf("a server made without a token has the token %q", got)
	}
}

func TestAnEmptyTokenLetsNoRequestThrough(t *testing.T) {
	t.Parallel()
	srv, url := newServedHandler(t, gardenToken)
	srv.SetToken("")

	for _, headers := range []map[string]string{
		nil,
		{"Authorization": "Bearer "},
		{"Authorization": "Bearer " + gardenToken},
	} {
		res := sendRequest(t, http.MethodPost, url+"/mcp", headers)
		assertUnauthorizedResponse(t, res, fmt.Sprintf("an initialize with the headers %v to a server with an empty token", headers))
	}
}

func TestSetTokenRejectsTheOldOneAndServesTheNewOne(t *testing.T) {
	t.Parallel()
	srv, url := newServedHandler(t, gardenToken)

	srv.SetToken("orchard-token-9021")

	assertUnauthorizedResponse(t, sendRequest(t, http.MethodPost, url+"/mcp",
		map[string]string{"Authorization": "Bearer " + gardenToken}), "an initialize with the old token")
	if res := sendRequest(t, http.MethodPost, url+"/mcp",
		map[string]string{"Authorization": "Bearer orchard-token-9021"}); res.StatusCode != http.StatusOK {
		t.Errorf("an initialize with the new token returned %d, want 200", res.StatusCode)
	}
}

func TestAnOptionsRequestWithoutTheTokenGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodOptions, url+"/mcp", map[string]string{
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "authorization, content-type",
	})

	assertUnauthorizedResponse(t, res, "an OPTIONS request without the token")
}

func TestNoResponseHasACORSHeader(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)
	authorized := map[string]string{"Authorization": "Bearer " + gardenToken}

	responses := map[string]httpResponse{
		"an initialize with the token":       sendRequest(t, http.MethodPost, url+"/mcp", authorized),
		"an initialize without the token":    sendRequest(t, http.MethodPost, url+"/mcp", nil),
		"an OPTIONS request without a token": sendRequest(t, http.MethodOptions, url+"/mcp", nil),
		"an OPTIONS request with the token":  sendRequest(t, http.MethodOptions, url+"/mcp", authorized),
	}

	for what, res := range responses {
		for key := range res.Header {
			if strings.HasPrefix(http.CanonicalHeaderKey(key), "Access-Control-Allow-") {
				t.Errorf("%s returned the header %s: %q", what, key, res.Header.Get(key))
			}
		}
	}
}

func TestARequestFromAnOriginOutsideTheMachineGets403(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	for _, origin := range []string{
		"null",
		"http://example.com",
		"https://example.com:3333",
		"http://127.0.0.1.example.com",
		"http://localhost.example.com",
		"http://192.168.1.20:3333",
		"ftp://localhost",
	} {
		withToken := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{
			"Origin": origin, "Authorization": "Bearer " + gardenToken,
		})
		if withToken.StatusCode != http.StatusForbidden {
			t.Errorf("an initialize with the token from the origin %s returned %d, want 403", origin, withToken.StatusCode)
		}
		withoutToken := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Origin": origin})
		if withoutToken.StatusCode != http.StatusForbidden {
			t.Errorf("an initialize without the token from the origin %s returned %d, want 403", origin, withoutToken.StatusCode)
		}
	}
}

func TestARequestFromALoopbackOriginIsCheckedForTheTokenOnly(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	for _, origin := range []string{
		"http://localhost",
		"http://localhost:5173",
		"https://localhost:8443",
		"http://127.0.0.1:3333",
		"http://127.8.9.10",
		"http://[::1]:3333",
	} {
		res := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{
			"Origin": origin, "Authorization": "Bearer " + gardenToken,
		})
		if res.StatusCode != http.StatusOK {
			t.Errorf("an initialize with the token from the origin %s returned %d, want 200", origin, res.StatusCode)
		}
		assertUnauthorizedResponse(t, sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Origin": origin}),
			"an initialize without the token from the origin "+origin)
	}
}
