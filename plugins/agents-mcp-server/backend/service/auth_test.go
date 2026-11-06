package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

const gardenToken = "garden-token-4417"

// newServedHandler serves the handler of a server whose token is token.
func newServedHandler(t *testing.T, token string) (*Server, string) {
	t.Helper()
	host := &instructedHost{listing: tools.Listing{Tools: []*tools.ToolInfo{newReadTool("orchard_pick")}}}
	srv, err := NewServer(Config{Token: token}, host)
	if err != nil {
		t.Fatal(err)
	}
	served := httptest.NewServer(srv.Handler())
	t.Cleanup(served.Close)
	return srv, served.URL
}

// sendRequest sends a request of the method to target with the headers and returns the answer with
// its body read.
func sendRequest(t *testing.T, method, target string, headers map[string]string) *http.Response {
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
	return res
}

func assertUnauthorizedAnswer(t *testing.T, res *http.Response, what string) {
	t.Helper()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("%s was answered %d, want 401", what, res.StatusCode)
	}
	if got := res.Header.Get("WWW-Authenticate"); got != `Bearer realm="numen"` {
		t.Errorf("%s was answered with WWW-Authenticate %q, want Bearer realm=\"numen\"", what, got)
	}
}

func TestARequestWithTheBearerTokenIsServed(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Authorization": "Bearer " + gardenToken})

	if res.StatusCode != http.StatusOK {
		t.Errorf("an initialize with the token was answered %d, want 200", res.StatusCode)
	}
}

func TestARequestWithoutTheTokenGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp", nil)

	assertUnauthorizedAnswer(t, res, "an initialize without the token")
}

func TestARequestWithAWrongTokenGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Authorization": "Bearer garden-token-4418"})

	assertUnauthorizedAnswer(t, res, "an initialize with a wrong token")
}

func TestATokenInTheQueryGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodPost, url+"/mcp?token="+gardenToken, nil)

	assertUnauthorizedAnswer(t, res, "an initialize with the token in the query")
}

func TestAServerMadeWithoutATokenHasAnEmptyToken(t *testing.T) {
	t.Parallel()

	srv, err := NewServer(Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}

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
		assertUnauthorizedAnswer(t, res, fmt.Sprintf("an initialize with the headers %v to a server with an empty token", headers))
	}
}

func TestASetTokenRefusesTheOldOneAndServesTheNewOne(t *testing.T) {
	t.Parallel()
	srv, url := newServedHandler(t, gardenToken)

	srv.SetToken("orchard-token-9021")

	assertUnauthorizedAnswer(t, sendRequest(t, http.MethodPost, url+"/mcp",
		map[string]string{"Authorization": "Bearer " + gardenToken}), "an initialize with the old token")
	if res := sendRequest(t, http.MethodPost, url+"/mcp",
		map[string]string{"Authorization": "Bearer orchard-token-9021"}); res.StatusCode != http.StatusOK {
		t.Errorf("an initialize with the new token was answered %d, want 200", res.StatusCode)
	}
}

func TestAnOptionsRequestWithoutTheTokenGets401(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)

	res := sendRequest(t, http.MethodOptions, url+"/mcp", map[string]string{
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "authorization, content-type",
	})

	assertUnauthorizedAnswer(t, res, "an OPTIONS request without the token")
}

func TestNoAnswerCarriesACORSHeader(t *testing.T) {
	t.Parallel()
	_, url := newServedHandler(t, gardenToken)
	authorized := map[string]string{"Authorization": "Bearer " + gardenToken}

	answers := map[string]*http.Response{
		"an initialize with the token":       sendRequest(t, http.MethodPost, url+"/mcp", authorized),
		"an initialize without the token":    sendRequest(t, http.MethodPost, url+"/mcp", nil),
		"an OPTIONS request without a token": sendRequest(t, http.MethodOptions, url+"/mcp", nil),
		"an OPTIONS request with the token":  sendRequest(t, http.MethodOptions, url+"/mcp", authorized),
	}

	for what, res := range answers {
		for key := range res.Header {
			if strings.HasPrefix(http.CanonicalHeaderKey(key), "Access-Control-Allow-") {
				t.Errorf("%s was answered with the header %s: %q", what, key, res.Header.Get(key))
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
			t.Errorf("an initialize with the token from the origin %s was answered %d, want 403", origin, withToken.StatusCode)
		}
		withoutToken := sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Origin": origin})
		if withoutToken.StatusCode != http.StatusForbidden {
			t.Errorf("an initialize without the token from the origin %s was answered %d, want 403", origin, withoutToken.StatusCode)
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
			t.Errorf("an initialize with the token from the origin %s was answered %d, want 200", origin, res.StatusCode)
		}
		assertUnauthorizedAnswer(t, sendRequest(t, http.MethodPost, url+"/mcp", map[string]string{"Origin": origin}),
			"an initialize without the token from the origin "+origin)
	}
}

func TestReadConfigReadsTheTokenValueOnly(t *testing.T) {
	t.Setenv("MCP_TOKEN", "env-token-5150")

	if cfg := readConfig(nil); cfg.Token != "" {
		t.Errorf("readConfig without values = %+v, want an empty token", cfg)
	}
	if cfg := readConfig(map[string]string{"port": "9002", "token": " map-token "}); cfg.Token != "map-token" {
		t.Errorf("readConfig = %+v, want the configured token trimmed", cfg)
	}
}
