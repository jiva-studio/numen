package mcpendpoint

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func dialEndpoint(t *testing.T, served *Endpoint) net.Conn {
	t.Helper()
	address, err := url.Parse(served.url)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", address.Host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Log(err)
		}
	})
	return conn
}

func waitFor(t *testing.T, isReady func() bool) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for !isReady() {
		select {
		case <-deadline:
			t.Fatal("the endpoint did not reach the expected state")
		case <-time.After(time.Millisecond):
		}
	}
}

func TestCloseDoesNotWaitForAConnectionThatSentNoRequest(t *testing.T) {
	served, err := serve(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	dialEndpoint(t, served)
	waitFor(t, func() bool {
		served.statesMu.Lock()
		defer served.statesMu.Unlock()
		return len(served.states) > 0
	})

	closed := make(chan error, 1)
	go func() { closed <- served.Close(t.Context()) }()
	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("close reported %v for a connection with no request", err)
		}
	case <-time.After(shutdownLimit / 2):
		t.Errorf("close took %v or longer", shutdownLimit/2)
	}
}

func TestCloseReportsAnInterruptedRequest(t *testing.T) {
	served, err := serve(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	served.limit = 100 * time.Millisecond
	conn := dialEndpoint(t, served)
	if _, err := conn.Write([]byte("POST /mcp HTTP/1.1\r\nHost: stand-in\r\nContent-Length: 100\r\n\r\n{")); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return served.getActiveCount() > 0 })

	if err := served.Close(t.Context()); err == nil {
		t.Error("close reported nothing for a request it cut")
	}
}

func TestTrackConnectionRemovesHijackedAndClosedConnections(t *testing.T) {
	served, err := serve(t.Context(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := served.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()
	if len(served.GetToken()) != tokenBytes*2 || served.GetURL() == "" {
		t.Errorf("token len = %d, url = %q", len(served.GetToken()), served.GetURL())
	}

	c1, c2 := net.Pipe()
	defer func() {
		if err := c1.Close(); err != nil {
			t.Error(err)
		}
		if err := c2.Close(); err != nil {
			t.Error(err)
		}
	}()

	served.trackConnection(c1, http.StateActive)
	if got := served.getActiveCount(); got != 1 {
		t.Errorf("active count = %d, want 1", got)
	}
	served.trackConnection(c1, http.StateHijacked)
	if got := served.getActiveCount(); got != 0 {
		t.Errorf("active count after hijack = %d, want 0", got)
	}
	served.trackConnection(c1, http.StateActive)
	served.trackConnection(c1, http.StateClosed)
	if got := served.getActiveCount(); got != 0 {
		t.Errorf("active count after close = %d, want 0", got)
	}
}

func TestServedToolsInvokeAndReportErrors(t *testing.T) {
	listed := []*tools.ToolInfo{
		{Name: "pond_read", Title: "Read pond", Effect: tools.ReadOnly, InputSchema: []byte(`{"type":"object"}`)},
		{Name: "pond_fail", Title: "Fail pond", Effect: tools.ReadOnly, InputSchema: []byte(`{"type":"object"}`)},
	}
	served, err := (Server{}).Serve(t.Context(), listed, func(_ context.Context, name string, args []byte) (*tools.CallResponse, error) {
		if name == "pond_fail" {
			return nil, errors.New("tool broke")
		}
		return &tools.CallResponse{Result: &tools.CallResult{Text: "got:" + string(args)}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := served.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", "Bearer "+served.GetToken())
		return http.DefaultTransport.RoundTrip(req)
	})}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, nil).
		Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: served.GetURL(), HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	}()

	okRes, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "pond_read", Arguments: map[string]any{"k": "v"}})
	if err != nil || okRes.IsError || len(okRes.Content) != 1 || getFirstText(okRes.Content) != `got:{"k":"v"}` {
		t.Errorf("pond_read = %+v, %v", okRes, err)
	}
	failRes, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "pond_fail"})
	if err != nil || !failRes.IsError || len(failRes.Content) != 1 || getFirstText(failRes.Content) != "tool broke" {
		t.Errorf("pond_fail = %+v, %v", failRes, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
