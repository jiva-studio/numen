package plugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type mockAgentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	tools        []*v1.AgentToolDefinition
	systemPrompt string
}

func (s *mockAgentServer) AskAgent(
	ctx context.Context,
	req *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	_ = stream.Send(&v1.AskAgentResponse{
		Step: &v1.AskAgentResponse_ModelStarted{
			ModelStarted: &v1.ModelStarted{},
		},
	})

	_ = stream.Send(&v1.AskAgentResponse{
		Step: &v1.AskAgentResponse_ToolCall{
			ToolCall: &v1.ToolCall{
				Tool:    "note_search",
				About:   "query",
				Written: 10,
				Path:    "notes/test.md",
				Span:    &v1.Span{From: 0, To: 100},
			},
		},
	})

	_ = stream.Send(&v1.AskAgentResponse{
		Step: &v1.AskAgentResponse_ToolFinished{
			ToolFinished: &v1.ToolFinished{},
		},
	})

	_ = stream.Send(&v1.AskAgentResponse{
		Step: &v1.AskAgentResponse_Said{
			Said: "answer: " + req.Msg.GetPrompt(),
		},
	})

	_ = stream.Send(&v1.AskAgentResponse{
		Step: &v1.AskAgentResponse_Stopped{
			Stopped: "done",
		},
	})

	return nil
}

func (s *mockAgentServer) FinishConversation(
	_ context.Context,
	_ *connect.Request[v1.FinishConversationRequest],
) (*connect.Response[v1.FinishConversationResponse], error) {
	return connect.NewResponse(&v1.FinishConversationResponse{}), nil
}

func (s *mockAgentServer) RegisterTool(
	_ context.Context,
	req *connect.Request[v1.RegisterToolRequest],
) (*connect.Response[v1.RegisterToolResponse], error) {
	s.tools = append(s.tools, req.Msg.GetTool())
	return connect.NewResponse(&v1.RegisterToolResponse{Success: true}), nil
}

func (s *mockAgentServer) ExtendSystemPrompt(
	_ context.Context,
	req *connect.Request[v1.ExtendSystemPromptRequest],
) (*connect.Response[v1.ExtendSystemPromptResponse], error) {
	s.systemPrompt += req.Msg.GetExtension()
	return connect.NewResponse(&v1.ExtendSystemPromptResponse{
		Success:    true,
		FullPrompt: s.systemPrompt,
	}), nil
}

func (s *mockAgentServer) GetAgentTools(
	_ context.Context,
	_ *connect.Request[v1.GetAgentToolsRequest],
) (*connect.Response[v1.GetAgentToolsResponse], error) {
	return connect.NewResponse(&v1.GetAgentToolsResponse{
		Tools:                 s.tools,
		SystemPromptExtension: s.systemPrompt,
	}), nil
}

func createTestAgentPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "agent")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.agent",
		"name": "Agent Plugin",
		"version": "1.0.0",
		"description": "Agent and chat provider",
		"capabilities": ["agent"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onCommand:askAgent"],
		"permissions": [],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/agent",
			"serviceName": "numen.v1.AgentService"
		},
		"agent": {
			"systemPrompt": "base prompt",
			"tools": [
				{
					"name": "manifest_tool",
					"description": "manifest tool description"
				}
			]
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestAgentBridge(t *testing.T) {
	serverImpl := &mockAgentServer{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(serverImpl))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestAgentPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	plugin, ok := host.GetPlugin("md.numen.agent")
	if !ok {
		t.Fatalf("plugin not found")
	}
	if plugin.Manifest.Agent.SystemPrompt != "base prompt" {
		t.Errorf("manifest system prompt = %q, want 'base prompt'", plugin.Manifest.Agent.SystemPrompt)
	}
	if len(plugin.Manifest.Agent.Tools) != 1 || plugin.Manifest.Agent.Tools[0].Name != "manifest_tool" {
		t.Errorf("manifest tools = %+v, want 1 tool", plugin.Manifest.Agent.Tools)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := OpenAgentBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenAgentBridge failed: %v", err)
	}

	run, err := bridge.StartRun(ctx, port.Task{
		Question: "hello world",
	})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	var steps []port.Step
	for step := range run.GetSteps() {
		steps = append(steps, step)
	}

	if len(steps) != 5 {
		t.Fatalf("got %d steps, want 5", len(steps))
	}
	if steps[0].Kind != domain.StepModelStart {
		t.Errorf("step 0 kind = %v, want StepModelStart", steps[0].Kind)
	}
	if steps[1].Kind != domain.StepToolCall || steps[1].Tool != "note_search" {
		t.Errorf("step 1 kind = %v, tool = %v", steps[1].Kind, steps[1].Tool)
	}
	if steps[2].Kind != domain.StepToolDone {
		t.Errorf("step 2 kind = %v, want StepToolDone", steps[2].Kind)
	}
	if steps[3].Kind != domain.StepSay || steps[3].Text != "answer: hello world" {
		t.Errorf("step 3 = %+v, want answer text", steps[3])
	}
	if steps[4].Kind != domain.StepStop || steps[4].Detail != "done" {
		t.Errorf("step 4 = %+v, want stop done", steps[4])
	}

	if err := bridge.EndConversation(ctx, "conv-1"); err != nil {
		t.Fatalf("EndConversation failed: %v", err)
	}

	if err := bridge.RegisterTool(ctx, "custom_calc", "calculate math", ""); err != nil {
		t.Fatalf("RegisterTool failed: %v", err)
	}

	fullPrompt, err := bridge.ExtendSystemPrompt(ctx, " extra instructions")
	if err != nil {
		t.Fatalf("ExtendSystemPrompt failed: %v", err)
	}
	if fullPrompt != " extra instructions" {
		t.Errorf("fullPrompt = %q", fullPrompt)
	}

	tools, promptExt, err := bridge.GetAgentTools(ctx)
	if err != nil {
		t.Fatalf("GetAgentTools failed: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "custom_calc" {
		t.Errorf("tools = %+v, want custom_calc", tools)
	}
	if promptExt != " extra instructions" {
		t.Errorf("promptExt = %q", promptExt)
	}

	_ = bridge.Close()
}

func TestAgentBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := NewHost("desktop.editor", tmp)
	_ = host.Discover()

	_, err := OpenAgentBridge(t.Context(), host)
	if err != ErrNoAgentPlugin {
		t.Fatalf("got err = %v, want ErrNoAgentPlugin", err)
	}
}

type floodAgentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
}

func (s *floodAgentServer) AskAgent(
	ctx context.Context,
	_ *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	for i := 0; i < 50; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_ = stream.Send(&v1.AskAgentResponse{
			Step: &v1.AskAgentResponse_Said{
				Said: fmt.Sprintf("msg %d", i),
			},
		})
	}
	return nil
}

func TestAgentBridge_UnreadChannelDoesNotDeadlock(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(&floodAgentServer{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestAgentPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := OpenAgentBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenAgentBridge failed: %v", err)
	}
	defer bridge.Close()

	run, err := bridge.StartRun(ctx, port.Task{Question: "test flood"})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	stopped := make(chan struct{})
	go func() {
		_ = run.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock in run.Stop() with unread steps channel")
	}
}

func TestConvertResponseStepCarriesTheCallNameAndEffect(t *testing.T) {
	effects := map[string]domain.StepKind{
		"read":   domain.StepRead,
		"edit":   domain.StepEdit,
		"remove": domain.StepRemove,
		"move":   domain.StepMove,
		"search": domain.StepSearch,
		"":       domain.StepToolCall,
		"other":  domain.StepToolCall,
	}
	for word, want := range effects {
		step := convertResponseStep(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_ToolCall{
			ToolCall: &v1.ToolCall{Tool: "Edit a note", Call: "toolu_01", Kind: word},
		}})
		if step.Kind != want || step.Call != "toolu_01" || step.Tool != "Edit a note" {
			t.Errorf("kind %q: got %+v", word, step)
		}
	}
}
