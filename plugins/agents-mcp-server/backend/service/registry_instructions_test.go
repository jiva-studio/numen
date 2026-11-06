package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

const (
	wantServerLine = "These tools work on one Numén vault: markdown notes in an ordinary folder, joined by links, with books, PDFs and recordings filed beside them."

	orchardInstructions = "Pick a fruit with orchard_pick before you name its tree, and link every tree you name."
	beehiveInstructions = "Count the bees with beehive_count, one frame at a time."
)

// instructedHost lists the tools and instructions it holds, which a test changes between sessions.
type instructedHost struct {
	mu      sync.Mutex
	listing tools.Listing
}

func (h *instructedHost) ListTools(context.Context) (tools.Listing, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.listing, nil
}

func (h *instructedHost) InvokeTool(context.Context, tools.Call) (*tools.CallResponse, error) {
	return &tools.CallResponse{Result: &tools.CallResult{Text: "picked"}}, nil
}

func (h *instructedHost) setListing(listing tools.Listing) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.listing = listing
}

// instructedHostHandler serves a listing as the host's PluginToolService does.
type instructedHostHandler struct {
	listing tools.Listing
}

func (h instructedHostHandler) ListTools(context.Context, http.Header) (tools.Listing, error) {
	return h.listing, nil
}

func (h instructedHostHandler) InvokeTool(context.Context, http.Header, tools.Call) (*tools.CallResponse, error) {
	return &tools.CallResponse{Result: &tools.CallResult{Text: "picked"}}, nil
}

func newReadTool(name string) *tools.ToolInfo {
	return &tools.ToolInfo{
		Name: name, Description: "Stand-in tool " + name, InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadOnly,
	}
}

// openSession connects a new MCP client to the server and returns the instructions of its
// initialize result.
func openSession(t *testing.T, srv *Server) string {
	t.Helper()
	serverSide, clientSide := sdk.NewInMemoryTransports()
	if _, err := srv.GetMCPServer().Connect(t.Context(), serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "orchard-client", Version: "1"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	cleanup.CloseAfter(t, session)
	return session.InitializeResult().Instructions
}

func TestTheServerShowsTheInstructionsOfAPluginRegistry(t *testing.T) {
	t.Parallel()
	host := &instructedHost{listing: tools.Listing{
		Tools:        []*tools.ToolInfo{newReadTool("orchard_pick")},
		Instructions: orchardInstructions,
	}}
	srv, err := NewServer(Config{}, host)
	if err != nil {
		t.Fatal(err)
	}

	instructions := openSession(t, srv)

	if want := wantServerLine + "\n\n" + orchardInstructions; instructions != want {
		t.Errorf("the instructions are\n%q\nwant\n%q", instructions, want)
	}
}

func TestTheServerKeepsTheJoinOfTheHost(t *testing.T) {
	t.Parallel()
	joined := beehiveInstructions + "\n\n" + orchardInstructions
	host := &instructedHost{listing: tools.Listing{
		Tools:        []*tools.ToolInfo{newReadTool("beehive_count"), newReadTool("orchard_pick")},
		Instructions: joined,
	}}
	srv, err := NewServer(Config{}, host)
	if err != nil {
		t.Fatal(err)
	}

	instructions := openSession(t, srv)

	if want := wantServerLine + "\n\n" + joined; instructions != want {
		t.Errorf("the instructions are\n%q\nwant\n%q", instructions, want)
	}
}

func TestDisablingThePluginRemovesItsInstructionsFromANewSession(t *testing.T) {
	t.Parallel()
	host := &instructedHost{listing: tools.Listing{
		Tools:        []*tools.ToolInfo{newReadTool("beehive_count"), newReadTool("orchard_pick")},
		Instructions: orchardInstructions,
	}}
	srv, err := NewServer(Config{}, host)
	if err != nil {
		t.Fatal(err)
	}
	if before := openSession(t, srv); !strings.Contains(before, orchardInstructions) {
		t.Fatalf("before the plugin is disabled the instructions are %q", before)
	}

	host.setListing(tools.Listing{Tools: []*tools.ToolInfo{newReadTool("beehive_count")}})
	instructions := openSession(t, srv)

	if strings.Contains(instructions, orchardInstructions) {
		t.Errorf("a session opened after the plugin is disabled is told %q", instructions)
	}
	if instructions != wantServerLine {
		t.Errorf("the instructions are %q, want %q", instructions, wantServerLine)
	}
}

func TestTheServerNamesNoToolOfAnotherPlugin(t *testing.T) {
	t.Parallel()
	names := []string{
		"library_search", "library_read", "note_search", "note_list", "file_read", "source_focus", "window_tabs_get",
	}
	toolInfos := make([]*tools.ToolInfo, 0, len(names))
	for _, name := range names {
		toolInfos = append(toolInfos, newReadTool(name))
	}
	srv, err := NewServer(Config{}, &instructedHost{listing: tools.Listing{Tools: toolInfos}})
	if err != nil {
		t.Fatal(err)
	}

	instructions := openSession(t, srv)

	for _, name := range names {
		if strings.Contains(instructions, name) {
			t.Errorf("the server's own instructions name the tool %s:\n%s", name, instructions)
		}
	}
	if instructions != wantServerLine {
		t.Errorf("the instructions are %q, want %q", instructions, wantServerLine)
	}
}

func TestTheInstructionsTheHostListsReachTheServerOverTheWire(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(tools.NewServiceHandler(instructedHostHandler{listing: tools.Listing{
		Tools:        []*tools.ToolInfo{newReadTool("orchard_pick")},
		Instructions: orchardInstructions,
	}}))
	served := httptest.NewServer(mux)
	t.Cleanup(served.Close)
	srv, err := NewServer(Config{}, tools.NewClient(hostcall.Endpoint{BaseURL: served.URL, Token: "orchard-token"}))
	if err != nil {
		t.Fatal(err)
	}

	instructions := openSession(t, srv)

	if want := wantServerLine + "\n\n" + orchardInstructions; instructions != want {
		t.Errorf("the instructions are\n%q\nwant\n%q", instructions, want)
	}
}
