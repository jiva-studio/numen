package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// standInHost serves PluginToolService as the host does: it lists the vault tools of the other
// plugins to the token it gave this plugin, and answers a call with the note it read.
type standInHost struct {
	mu      sync.Mutex
	headers []string
	calls   []tools.Call
}

func (h *standInHost) ListToolInfos(_ context.Context, header http.Header) ([]*tools.ToolInfo, error) {
	h.record(header, nil)
	return []*tools.ToolInfo{{
		Name: "note_read", Title: "Read a note", Effect: tools.ReadOnly,
		InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}}, nil
}

func (h *standInHost) InvokeTool(_ context.Context, header http.Header, call tools.Call) (*tools.CallResponse, error) {
	h.record(header, &call)
	return &tools.CallResponse{Result: &tools.CallResult{Text: "# Garden\nRaised beds."}}, nil
}

func (h *standInHost) record(header http.Header, call *tools.Call) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.headers = append(h.headers, header.Get("Authorization"))
	if call != nil {
		h.calls = append(h.calls, *call)
	}
}

// A program outside Numen lists and calls a vault tool, which the plugin reaches through the host
// under its own token, with no context.
func TestAProgramCallsAVaultToolThroughTheHost(t *testing.T) {
	host := &standInHost{}
	mux := http.NewServeMux()
	mux.Handle(tools.NewServiceHandler(host))
	served := httptest.NewServer(mux)
	t.Cleanup(served.Close)
	endpoint := hostcall.Endpoint{BaseURL: served.URL, Token: "plugin-token"}
	srv, err := NewServer(Config{}, tools.NewClient(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	serverSide, clientSide := sdk.NewInMemoryTransports()
	if _, err := srv.GetMCPServer().Connect(t.Context(), serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "program"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 1 || listed.Tools[0].Name != "note_read" {
		t.Fatalf("the program was offered %v", listed.Tools)
	}
	result, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "note_read", Arguments: json.RawMessage(`{"path":"Garden.md"}`),
		Meta: sdk.Meta{"md.example.cards/card": "k7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text, isText := result.Content[0].(*sdk.TextContent); !isText || text.Text != "# Garden\nRaised beds." {
		t.Errorf("the call answered %v", result.Content)
	}

	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.calls) != 1 || host.calls[0].GetName() != "note_read" || string(host.calls[0].GetArguments()) != `{"path":"Garden.md"}` ||
		len(host.calls[0].GetContext()) != 0 {
		t.Errorf("the host was called with %v", host.calls)
	}
	for _, header := range host.headers {
		if header != "Bearer plugin-token" {
			t.Errorf("the host was called with %q", header)
		}
	}
}
