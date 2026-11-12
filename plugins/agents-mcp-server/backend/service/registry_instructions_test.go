package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

const (
	wantServerLine = "These tools work on one Numén vault: markdown notes in a folder, connected by links, and the books, PDFs and recordings stored in the same folder."

	orchardPickSentence  = "Pick a fruit with orchard_pick before you name its tree."
	orchardShareSentence = "Link every tree you name, whether orchard_pick or orchard_graft found it."
	orchardGraftSentence = "Graft a branch with orchard_graft only onto a tree you have picked from."
	beehiveCountSentence = "Count the bees with beehive_count, one frame at a time."
)

// fakeToolHost lists the tools and instructions it has, which a test changes between sessions.
type fakeToolHost struct {
	mu      sync.Mutex
	listing tools.Listing
}

func (h *fakeToolHost) ListTools(context.Context) (tools.Listing, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.listing, nil
}

func (h *fakeToolHost) InvokeTool(context.Context, tools.Call) (*tools.CallResponse, error) {
	return &tools.CallResponse{Result: &tools.CallResult{Text: "picked"}}, nil
}

func (h *fakeToolHost) setListing(listing tools.Listing) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.listing = listing
}

// fakeToolHostHandler serves a listing as the host's PluginToolService does.
type fakeToolHostHandler struct {
	listing tools.Listing
}

func (h fakeToolHostHandler) ListTools(context.Context, http.Header) (tools.Listing, error) {
	return h.listing, nil
}

func (h fakeToolHostHandler) InvokeTool(context.Context, http.Header, tools.Call) (*tools.CallResponse, error) {
	return &tools.CallResponse{Result: &tools.CallResult{Text: "picked"}}, nil
}

func newReadTool(name string) *tools.ToolInfo {
	return &tools.ToolInfo{
		Name: name, Description: "Stand-in tool " + name, InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadOnly,
	}
}

func openSession(t *testing.T, srv *testServer) string {
	t.Helper()
	serverSide, clientSide := sdk.NewInMemoryTransports()
	if _, err := srv.GetMCPServer().Connect(t.Context(), serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "orchard-client", Version: "1"}, nil).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeAndCheck(t, session) })
	return session.InitializeResult().Instructions
}

func newToolWithSentences(name string, sentences ...string) *tools.ToolInfo {
	tool := newReadTool(name)
	tool.Instructions = sentences
	return tool
}

func TestServerSendsDistinctToolSentencesInListingOrder(t *testing.T) {
	t.Parallel()
	host := &fakeToolHost{listing: tools.Listing{Tools: []*tools.ToolInfo{
		newToolWithSentences("beehive_count", beehiveCountSentence),
		newToolWithSentences("orchard_pick", orchardPickSentence, orchardShareSentence),
		newToolWithSentences("orchard_graft", orchardGraftSentence, orchardShareSentence),
		newReadTool("orchard_rest"),
	}}}
	srv, err := NewServer(Config{}, host)
	if err != nil {
		t.Fatal(err)
	}

	instructions := openSession(t, srv)

	want := wantServerLine + "\n\n" + beehiveCountSentence + " " + orchardPickSentence + " " + orchardShareSentence + " " + orchardGraftSentence
	if instructions != want {
		t.Errorf("the instructions are\n%q\nwant\n%q", instructions, want)
	}
}

func TestSentenceOfUnlistedToolIsNotSentToANewSession(t *testing.T) {
	t.Parallel()
	host := &fakeToolHost{listing: tools.Listing{Tools: []*tools.ToolInfo{
		newToolWithSentences("beehive_count", beehiveCountSentence),
		newToolWithSentences("orchard_pick", orchardPickSentence),
	}}}
	srv, err := NewServer(Config{}, host)
	if err != nil {
		t.Fatal(err)
	}
	if before := openSession(t, srv); !strings.Contains(before, orchardPickSentence) {
		t.Fatalf("before the plugin is disabled the instructions are %q", before)
	}

	host.setListing(tools.Listing{Tools: []*tools.ToolInfo{newToolWithSentences("beehive_count", beehiveCountSentence)}})
	instructions := openSession(t, srv)

	if strings.Contains(instructions, orchardPickSentence) {
		t.Errorf("a session opened after the plugin is disabled is told %q", instructions)
	}
	if want := wantServerLine + "\n\n" + beehiveCountSentence; instructions != want {
		t.Errorf("the instructions are %q, want %q", instructions, want)
	}
}

func TestToolsWithNoSentencesLeaveOnlyTheServerLine(t *testing.T) {
	t.Parallel()
	names := []string{
		"library_search", "library_read", "note_search", "note_list", "file_read", "source_focus", "window_tab_list",
	}
	toolInfos := make([]*tools.ToolInfo, 0, len(names))
	for _, name := range names {
		toolInfos = append(toolInfos, newReadTool(name))
	}
	srv, err := NewServer(Config{}, &fakeToolHost{listing: tools.Listing{Tools: toolInfos}})
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

func TestHostListedSentencesAreSentToTheServerOverTheWire(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.Handle(tools.NewServiceHandler(fakeToolHostHandler{listing: tools.Listing{
		Tools: []*tools.ToolInfo{newToolWithSentences("orchard_pick", orchardPickSentence)},
	}}))
	served := httptest.NewServer(mux)
	t.Cleanup(served.Close)
	srv, err := NewServer(Config{}, tools.NewClient(hostcall.Endpoint{BaseURL: served.URL, Token: "orchard-token"}))
	if err != nil {
		t.Fatal(err)
	}

	instructions := openSession(t, srv)

	if want := wantServerLine + "\n\n" + orchardPickSentence; instructions != want {
		t.Errorf("the instructions are\n%q\nwant\n%q", instructions, want)
	}
}
