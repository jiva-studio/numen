package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

const initializeParams = `{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"orchard","version":"1"}}`

func sendInitialize(t *testing.T, conn sdk.Connection, number int) *jsonrpc.Response {
	t.Helper()
	id, err := jsonrpc.MakeID(float64(number))
	if err != nil {
		t.Fatal(err)
	}
	request := &jsonrpc.Request{ID: id, Method: "initialize", Params: json.RawMessage(initializeParams)}
	if err := conn.Write(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	msg, err := conn.Read(t.Context())
	if err != nil {
		t.Fatalf("initialize %d: %v", number, err)
	}
	res, ok := msg.(*jsonrpc.Response)
	if !ok {
		t.Fatalf("initialize %d returned %T", number, msg)
	}
	return res
}

func TestSecondInitializeReturnsAnError(t *testing.T) {
	t.Parallel()
	host := &fakeToolHost{listing: tools.Listing{
		Tools: []*tools.ToolInfo{newToolWithSentences("orchard_pick", orchardPickSentence)},
	}}
	srv, err := NewServer(Config{}, host)
	if err != nil {
		t.Fatal(err)
	}
	serverSide, clientSide := sdk.NewInMemoryTransports()
	if _, err := srv.GetMCPServer().Connect(t.Context(), serverSide, nil); err != nil {
		t.Fatal(err)
	}
	conn, err := clientSide.Connect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})

	first := sendInitialize(t, conn, 1)
	second := sendInitialize(t, conn, 2)

	if first.Error != nil || !strings.Contains(string(first.Result), "orchard_pick") {
		t.Errorf("the first initialize is returned %s, error %v", first.Result, first.Error)
	}
	if second.Error == nil {
		t.Errorf("a second initialize is returned %s with no error", second.Result)
	}
}

func postInitialize(t *testing.T, url string, number int, session string) (status int, body, sessionID string) {
	t.Helper()
	payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"initialize","params":%s}`, number, initializeParams)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url+"/mcp", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer orchard-token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(out), res.Header.Get("Mcp-Session-Id")
}

func TestSecondInitializeOverHTTPReturnsAnError(t *testing.T) {
	t.Parallel()
	host := &fakeToolHost{listing: tools.Listing{Tools: []*tools.ToolInfo{newReadTool("orchard_pick")}}}
	srv, err := NewServer(Config{Token: "orchard-token"}, host)
	if err != nil {
		t.Fatal(err)
	}
	served := httptest.NewServer(srv.Handler())
	t.Cleanup(served.Close)

	_, _, session := postInitialize(t, served.URL, 1, "")
	if session == "" {
		t.Fatal("the first initialize opened no session")
	}
	status, body, _ := postInitialize(t, served.URL, 2, session)

	if status != http.StatusOK || !strings.Contains(body, `"error"`) {
		t.Errorf("a second initialize is returned %d %s", status, body)
	}
}
