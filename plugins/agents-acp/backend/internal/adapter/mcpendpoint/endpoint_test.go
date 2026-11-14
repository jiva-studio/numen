package mcpendpoint

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"slices"
	"sync"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestServeDefaultsToolsAndTracksConnections(t *testing.T) {
	if shutdownLimit != 2*time.Second || shutdownPoll != 10*time.Millisecond {
		t.Fatalf("constants = (%v, %v), want (2s, 10ms)", shutdownLimit, shutdownPoll)
	}

	listed := []*tools.ToolInfo{
		{Name: "first_tool", InputSchema: []byte(`{"type":"object"}`)},
		{Name: "second_tool", InputSchema: []byte(`{"type":"object"}`)},
	}
	servedIfc, err := (Server{}).Serve(t.Context(), listed, func(_ context.Context, name string, _ []byte) (*tools.CallResponse, error) {
		return &tools.CallResponse{Result: &tools.CallResult{Text: "called:" + name}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := servedIfc.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	served, ok := servedIfc.(*Endpoint)
	if !ok {
		t.Fatalf("Serve returned %T", servedIfc)
	}
	if served.limit != 2*time.Second || served.server.ReadHeaderTimeout != 10*time.Second || served.GetToken() == "" {
		t.Fatalf("endpoint settings = (limit=%v, headerTimeout=%v, token=%q)", served.limit, served.server.ReadHeaderTimeout, served.GetToken())
	}

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", "Bearer "+served.GetToken())
		return http.DefaultTransport.RoundTrip(req)
	})}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "1"}, nil).
		Connect(t.Context(), &sdk.StreamableClientTransport{Endpoint: served.GetURL(), HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	}()
	toolsRes, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range toolsRes.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"first_tool", "second_tool"}) {
		t.Fatalf("listed tools = %v, want [first_tool second_tool]", names)
	}

	c1, c2 := net.Pipe()
	t.Cleanup(func() {
		if err := c1.Close(); err != nil {
			t.Error(err)
		}
		if err := c2.Close(); err != nil {
			t.Error(err)
		}
	})
	// The session above keeps its own connection tracked, so the counts are relative to it.
	active, tracked := served.getActiveCount(), getTrackedCount(served)
	served.trackConnection(c1, http.StateActive)
	if served.getActiveCount() != active+1 || getTrackedCount(served) != tracked+1 {
		t.Fatalf("after StateActive: active=%d states=%d, want %d, %d", served.getActiveCount(), getTrackedCount(served), active+1, tracked+1)
	}
	served.trackConnection(c1, http.StateClosed)
	if served.getActiveCount() != active || getTrackedCount(served) != tracked {
		t.Fatalf("after StateClosed: active=%d states=%d, want %d, %d", served.getActiveCount(), getTrackedCount(served), active, tracked)
	}
	served.trackConnection(c1, http.StateActive)
	served.trackConnection(c1, http.StateHijacked)
	if served.getActiveCount() != active || getTrackedCount(served) != tracked {
		t.Fatalf("after StateHijacked: active=%d states=%d, want %d, %d", served.getActiveCount(), getTrackedCount(served), active, tracked)
	}
}

func getTrackedCount(served *Endpoint) int {
	served.statesMu.Lock()
	defer served.statesMu.Unlock()
	return len(served.states)
}

type failingAcceptListener struct {
	err error
}

func (l failingAcceptListener) Accept() (net.Conn, error) {
	time.Sleep(20 * time.Millisecond)
	return nil, l.err
}

func (failingAcceptListener) Close() error   { return nil }
func (failingAcceptListener) Addr() net.Addr { return &net.TCPAddr{} }

type closeErrorListener struct {
	net.Listener
	closeErr error
}

func (l closeErrorListener) Close() error {
	return errors.Join(l.Listener.Close(), l.closeErr)
}

func TestCloseJoinsServeErrorAndShutdownCloseError(t *testing.T) {
	errAccept := errors.New("accept failed")
	ep := &Endpoint{
		server:  &http.Server{ReadHeaderTimeout: time.Second},
		stopped: make(chan struct{}),
		limit:   shutdownLimit,
		states:  make(map[net.Conn]http.ConnState),
	}
	go ep.run(failingAcceptListener{err: errAccept})
	// Serve keeps an accept error only when it ends before Close starts the shutdown.
	<-ep.stopped
	if err := ep.Close(t.Context()); !errors.Is(err, errAccept) {
		t.Fatalf("Close with failing listener = %v, want %v", err, errAccept)
	}

	rawLn, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	errClose := errors.New("listener close failed")
	epClose := &Endpoint{
		url:      "http://" + rawLn.Addr().String() + endpointPath,
		server:   &http.Server{Handler: http.NotFoundHandler(), ReadHeaderTimeout: time.Second},
		stopped:  make(chan struct{}),
		limit:    shutdownLimit,
		statesMu: sync.Mutex{},
		states:   make(map[net.Conn]http.ConnState),
	}
	epClose.server.ConnState = epClose.trackConnection
	go epClose.run(closeErrorListener{Listener: rawLn, closeErr: errClose})

	dialEndpoint(t, epClose)
	waitFor(t, func() bool {
		epClose.statesMu.Lock()
		defer epClose.statesMu.Unlock()
		return len(epClose.states) > 0
	})

	if err := epClose.Close(t.Context()); !errors.Is(err, errClose) {
		t.Fatalf("Close with listener close error = %v, want %v", err, errClose)
	}
}
