package service

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

type fakeToolClient struct {
	mu       sync.Mutex
	tools    []*tools.ToolInfo
	lastCall *tools.Call
	callResp *tools.CallResponse
	callErr  error
	listErr  error
}

func (f *fakeToolClient) ListTools(_ context.Context) (tools.Listing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return tools.Listing{}, f.listErr
	}
	return tools.Listing{Tools: f.tools}, nil
}

func (f *fakeToolClient) InvokeTool(_ context.Context, call tools.Call) (*tools.CallResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCall = &call
	if f.callErr != nil {
		return nil, f.callErr
	}
	if f.callResp != nil {
		return f.callResp, nil
	}
	return &tools.CallResponse{
		Result: &tools.CallResult{
			Text: "default answer",
		},
	}, nil
}
func connectInMemory(t *testing.T, srv *testServer, opts *sdk.ClientOptions) *sdk.ClientSession {
	t.Helper()
	serverSide, clientSide := sdk.NewInMemoryTransports()
	if _, err := srv.GetMCPServer().Connect(t.Context(), serverSide, nil); err != nil {
		t.Fatal(err)
	}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "pond-client"}, opts).Connect(t.Context(), clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeAndCheck(t, session) })
	return session
}

// listChangedDelay is how long a test waits for a notifications/tools/list_changed. go-sdk sends one
// 10ms after the last change of the tools.
const listChangedDelay = 500 * time.Millisecond

func connectCountingNotifications(t *testing.T, srv *testServer) (*sdk.ClientSession, <-chan struct{}) {
	t.Helper()
	notifications := make(chan struct{}, 64)
	opts := &sdk.ClientOptions{
		ToolListChangedHandler: func(context.Context, *sdk.ToolListChangedRequest) {
			select {
			case notifications <- struct{}{}:
			default:
			}
		},
	}
	return connectInMemory(t, srv, opts), notifications
}

func listTools(t *testing.T, session *sdk.ClientSession) []*sdk.Tool {
	t.Helper()
	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return listed.Tools
}

func waitForNotification(t *testing.T, notifications <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-notifications:
	case <-time.After(5 * time.Second):
		t.Fatalf("no notifications/tools/list_changed arrived after %s", what)
	}
}

func drainNotifications(notifications <-chan struct{}) {
	for {
		select {
		case <-notifications:
		case <-time.After(listChangedDelay):
			return
		}
	}
}

func newPondClient() *fakeToolClient {
	return &fakeToolClient{tools: []*tools.ToolInfo{
		{Name: "pond_read", Description: "Reads a pond note", InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadOnly},
		{Name: "pond_write", Description: "Writes a pond note", InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadWrite},
	}}
}

func TestListingUnchangedToolsSendsNoListChangedNotification(t *testing.T) {
	t.Parallel()
	srv, err := NewServer(Config{}, newPondClient())
	if err != nil {
		t.Fatal(err)
	}
	session, notifications := connectCountingNotifications(t, srv)

	for range 3 {
		listTools(t, session)
	}

	select {
	case <-notifications:
		t.Error("a notifications/tools/list_changed arrived though the host listed the same tools")
	case <-time.After(listChangedDelay):
	}
}

func TestAChangedDescriptionSendsAListChangedNotification(t *testing.T) {
	t.Parallel()
	client := newPondClient()
	srv, err := NewServer(Config{}, client)
	if err != nil {
		t.Fatal(err)
	}
	session, notifications := connectCountingNotifications(t, srv)
	listTools(t, session)
	drainNotifications(notifications)

	client.mu.Lock()
	client.tools[0] = &tools.ToolInfo{Name: "pond_read", Description: "Reads a pond note and its margins", InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadOnly}
	client.mu.Unlock()
	toolList := listTools(t, session)

	waitForNotification(t, notifications, "a changed description")
	if index := slices.IndexFunc(toolList, func(tool *sdk.Tool) bool { return tool.Name == "pond_read" }); index < 0 ||
		toolList[index].Description != "Reads a pond note and its margins" {
		t.Errorf("tools/list returned %v, want pond_read with the changed description", toolList)
	}
}

func TestAnAddedToolSendsAListChangedNotification(t *testing.T) {
	t.Parallel()
	client := newPondClient()
	srv, err := NewServer(Config{}, client)
	if err != nil {
		t.Fatal(err)
	}
	session, notifications := connectCountingNotifications(t, srv)
	listTools(t, session)
	drainNotifications(notifications)

	client.mu.Lock()
	client.tools = append(client.tools, &tools.ToolInfo{Name: "pond_fill", Description: "Fills a pond", InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadWrite})
	client.mu.Unlock()
	toolList := listTools(t, session)

	waitForNotification(t, notifications, "an added tool")
	if len(toolList) != 3 {
		t.Errorf("tools/list returned %d tools, want 3", len(toolList))
	}
}

func TestARemovedToolSendsAListChangedNotification(t *testing.T) {
	t.Parallel()
	client := newPondClient()
	srv, err := NewServer(Config{}, client)
	if err != nil {
		t.Fatal(err)
	}
	session, notifications := connectCountingNotifications(t, srv)
	listTools(t, session)
	drainNotifications(notifications)

	client.mu.Lock()
	client.tools = client.tools[:1]
	client.mu.Unlock()
	toolList := listTools(t, session)

	waitForNotification(t, notifications, "a removed tool")
	if len(toolList) != 1 || toolList[0].Name != "pond_read" {
		t.Errorf("tools/list returned %v, want pond_read alone", toolList)
	}
}
