package plugintest

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"connectrpc.com/connect"
	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

func CreateTestPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "sample")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "org.example.sample",
		"name": "Sample Plugin",
		"version": "1.0.0",
		"description": "Sample",
		"capabilities": ["org.example.reading"],
		"targets": ["desktop.editor"],
		"activationEvents": [],
		"contentTypes": [{"type": "application/x-sample", "globs": ["*.sample"]}],
		"texts": {"contentTypes": ["application/x-sample"]},
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/sample",
			"serviceName": "numen.v1.TextProviderService"
		},
		"frontend": {
			"entrypoint": "dist/index.js",
			"styles": "dist/styles.css"
		},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func KillProcess(t *testing.T, p *os.Process) {
	t.Helper()
	if err := p.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("kill process: %v", err)
	}
}

// ReapProcess waits for a killed process to exit. A process that the host has already waited for is not an error.
func ReapProcess(t *testing.T, p *os.Process) {
	t.Helper()
	if _, err := p.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ECHILD) {
		t.Errorf("wait for process: %v", err)
	}
}

func WaitUntil(t *testing.T, what string, condition func() bool) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for !condition() {
		select {
		case <-timeout.C:
			t.Fatalf("timed out waiting for %s", what)
		case <-ticker.C:
		}
	}
}

const helperProcessEnv = "NUMEN_TEST_HELPER_PROCESS"

// NewIdleCommand returns a command that runs a child process doing nothing until it is killed.
func NewIdleCommand(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcessIdles$")
	cmd.Env = append(os.Environ(), helperProcessEnv+"=idle")
	return cmd
}

// IdleChild is the body of the child process newIdleCommand starts; it does nothing in a normal run.
func IdleChild(t *testing.T) {
	if os.Getenv(helperProcessEnv) != "idle" {
		return
	}
	time.Sleep(time.Minute)
}

// protocolLifecycle answers Ping with the protocol it holds.
type protocolLifecycle struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	protocol int32
}

func (p protocolLifecycle) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Status: "ok", Protocol: p.protocol}), nil
}

// ServePing starts a server answering Ping with the given protocol and returns its port.
func ServePing(t *testing.T, protocol int32) int {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(protocolLifecycle{protocol: protocol}))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	addr, ok := srv.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected *net.TCPAddr, got %T", srv.Listener.Addr())
	}
	return addr.Port
}

// ServeHostPing starts a server answering Ping with the host's protocol and returns its port.
func ServeHostPing(t *testing.T) int {
	t.Helper()
	return ServePing(t, pluginport.PluginProtocolVersion)
}

type MockAgentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	tools        []*v1.AgentToolDefinition
	systemPrompt string
}

func (s *MockAgentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	steps := []*v1.AskAgentResponse{
		{Step: &v1.AskAgentResponse_ModelStarted{ModelStarted: &v1.ModelStarted{}}},
		{Step: &v1.AskAgentResponse_ToolCall{
			ToolCall: &v1.ToolCall{
				Tool:   "note_search",
				About:  "query",
				Length: 10,
				Path:   "notes/test.md",
				Range:  &v1.Range{Start: 0, End: 100},
			},
		}},
		{Step: &v1.AskAgentResponse_ToolFinished{ToolFinished: &v1.ToolFinished{}}},
		{Step: &v1.AskAgentResponse_Text{Text: "answer: " + req.Msg.GetPrompt()}},
		{Step: &v1.AskAgentResponse_Stopped{Stopped: "done"}},
	}
	for _, step := range steps {
		if err := stream.Send(step); err != nil {
			return err
		}
	}

	return nil
}

func (s *MockAgentServer) FinishConversation(
	_ context.Context,
	_ *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

func (s *MockAgentServer) RegisterTool(
	_ context.Context,
	req *connect.Request[v1.RegisterToolRequest],
) (*connect.Response[v1.RegisterToolResponse], error) {
	s.tools = append(s.tools, req.Msg.GetTool())
	return connect.NewResponse(&v1.RegisterToolResponse{Success: true}), nil
}

func (s *MockAgentServer) ExtendSystemPrompt(
	_ context.Context,
	req *connect.Request[v1.ExtendSystemPromptRequest],
) (*connect.Response[v1.ExtendSystemPromptResponse], error) {
	s.systemPrompt += req.Msg.GetExtension()
	return connect.NewResponse(&v1.ExtendSystemPromptResponse{
		Success:    true,
		FullPrompt: s.systemPrompt,
	}), nil
}

func (s *MockAgentServer) GetAgentTools(
	_ context.Context,
	_ *connect.Request[v1.GetAgentToolsRequest],
) (*connect.Response[v1.GetAgentToolsResponse], error) {
	return connect.NewResponse(&v1.GetAgentToolsResponse{
		Tools:                 s.tools,
		SystemPromptExtension: s.systemPrompt,
	}), nil
}

// LeaseHost resolves every capability to one stub plugin and counts the references taken and given back.
type LeaseHost struct {
	Endpoint pluginport.Endpoint
	// ensureErr is what activating the plugin fails with.
	EnsureErr error
	Acquired  atomic.Int32
	Released  atomic.Int32
}

func (h *LeaseHost) ResolveCapability(string) (pluginport.DiscoveredPlugin, error) {
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.stub"}}, nil
}

func (h *LeaseHost) ResolveService(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{}, false
}

func (h *LeaseHost) GetPlugin(string) (pluginport.DiscoveredPlugin, bool) {
	return pluginport.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.stub"}}, true
}

func (h *LeaseHost) EnsureActive(context.Context, string) (pluginport.Endpoint, error) {
	if h.EnsureErr != nil {
		return pluginport.Endpoint{}, h.EnsureErr
	}
	h.Acquired.Add(1)
	return h.Endpoint, nil
}

func (h *LeaseHost) Acquire(string) { h.Acquired.Add(1) }

func (h *LeaseHost) Release(string) { h.Released.Add(1) }

func (h *LeaseHost) CheckPluginPermission(_, _ string) error { return nil }

func (h *LeaseHost) GetVaultPath() string { return "" }

func (h *LeaseHost) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

type CompleterFunc func(ctx context.Context, question completionport.Question) (string, error)

func (f CompleterFunc) Complete(ctx context.Context, _ string, question completionport.Question) (string, error) {
	return f(ctx, question)
}

// GetPort returns the port the test server listens on.
func GetPort(tb testing.TB, server *httptest.Server) int {
	tb.Helper()
	addr, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		tb.Fatalf("test server listens on %T, want a TCP address", server.Listener.Addr())
	}
	return addr.Port
}
