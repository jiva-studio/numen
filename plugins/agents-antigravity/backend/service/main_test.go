package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp/acptest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools/toolstest"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

// TestMain runs the test binary as a fake Antigravity when it is started as one, and the tests of m
// otherwise.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "fake-agy" {
		time.AfterFunc(20*time.Second, func() { os.Exit(3) })
		os.Exit(runFakeAntigravity(os.Stdout))
	}
	os.Exit(m.Run())
}

// runFakeAntigravity calls note_read on the server of tools its home names, and writes what it did as
// Antigravity does.
func runFakeAntigravity(out io.Writer) int {
	answer, err := callNoteRead(filepath.Join(os.Getenv("HOME"), ".gemini", "config", "mcp_config.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	lines := []string{
		`{"event":"init","conversation_id":"s-garden"}`,
		`{"tool_calls":[{"name":"note_read","arguments":{"path":"Garden/Beds.md"}}]}`,
		`{"event":"step_update","step_update":{"step_type":"agent_response","text_delta":` + quoteJSON(strings.TrimPrefix(answer, "# Garden\n")) + `}}`,
		`{"event":"result","result":{"status":"SUCCESS","response":"Raised beds."}}`,
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return 1
		}
	}
	return 0
}

func quoteJSON(text string) string {
	raw, err := json.Marshal(text)
	if err != nil {
		return `""`
	}
	return string(raw)
}

func callNoteRead(configPath string) (string, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	var config struct {
		Servers map[string]struct {
			URL     string            `json:"serverUrl"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", err
	}
	server := config.Servers["numen"]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", server.Headers["Authorization"])
		return http.DefaultTransport.RoundTrip(req)
	})}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "fake-agy", Version: "1"}, nil).
		Connect(ctx, &sdk.StreamableClientTransport{Endpoint: server.URL, HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = session.Close() }()
	result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "note_read", Arguments: map[string]any{"path": "Garden/Beds.md"}})
	if err != nil {
		return "", err
	}
	var text string
	for _, content := range result.Content {
		if one, isText := content.(*sdk.TextContent); isText {
			text += one.Text
		}
	}
	return text, nil
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// fakeAntigravity returns a program that runs the test binary as a fake Antigravity.
func fakeAntigravity(t *testing.T) string {
	t.Helper()
	return writeScript(t, "agy", "exec "+quoteShell(os.Args[0])+" fake-agy\n")
}

func quoteShell(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

// gardenHost is a stand-in for the host that lists two tools of a garden vault and records the calls it
// runs.
type gardenHost struct {
	mu    sync.Mutex
	calls []tools.Call
}

func (h *gardenHost) ListTools(context.Context, http.Header) (tools.Listing, error) {
	return tools.Listing{Tools: []*tools.ToolInfo{
		{
			Name: "note_read", Title: "Read a note", Effect: tools.ReadOnly,
			InputSchema:  []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
			Instructions: []string{"Cite the note you read."},
		},
		{
			Name: "garden_plan", Title: "Plan the garden", Effect: tools.ReadWrite,
			InputSchema:  []byte(`{"type":"object"}`),
			Instructions: []string{"Cite the note you read.", "Ask before you plan."},
		},
	}}, nil
}

func (h *gardenHost) InvokeTool(_ context.Context, _ http.Header, call tools.Call) (*tools.CallResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, call)
	return &tools.CallResponse{Result: &tools.CallResult{Text: "# Garden\nRaised beds."}}, nil
}

func (h *gardenHost) listCalls() []tools.Call {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.calls)
}

func serveGardenHost(t *testing.T) (*gardenHost, tools.Client) {
	t.Helper()
	host := &gardenHost{}
	mux := http.NewServeMux()
	mux.Handle(toolstest.NewServiceHandler(host))
	served := httptest.NewServer(mux)
	t.Cleanup(served.Close)
	return host, tools.NewClient(hostcall.Endpoint{BaseURL: served.URL, Token: "orchard-token"})
}

func newTestAgent(t *testing.T, host task.Tools) *agent {
	t.Helper()
	built, err := newAgent(parts{tools: host, isAsync: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := built.tasks.Close(); err != nil {
			t.Error(err)
		}
	})
	return built
}

// newAntigravity returns the plugin running the program, in a home of the test's own.
func newAntigravity(t *testing.T, host task.Tools, program string) *agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	built := newTestAgent(t, host)
	built.settings.Apply(map[string]string{"executablePath": program})
	return built
}

func agentCall(options map[string]string) connection.Call {
	return connection.Call{ConnectionID: "conn-orchard", Type: "antigravity", Options: options}
}

func runTask(ctx context.Context, svc *agent, call connection.Call, request *acp.TaskRequest) ([]string, *acp.TaskDone, error) {
	var frames []string
	done, err := acptest.RunTask(ctx, svc.Provider(), call, request, func(frame *acp.TaskFrame) error {
		frames = append(frames, describeFrame(frame))
		return nil
	})
	return frames, done, err
}

func describeFrame(frame *acp.TaskFrame) string {
	switch update := frame.GetUpdate().(type) {
	case *acp.TaskFrame_AgentMessageChunk:
		return fmt.Sprintf("chunk %s %q", update.AgentMessageChunk.GetMessageId(), update.AgentMessageChunk.GetText())
	case *acp.TaskFrame_ToolCall:
		call := update.ToolCall
		var paths []string
		for _, one := range call.GetLocations() {
			paths = append(paths, one.GetPath())
		}
		return fmt.Sprintf("call %s %q %s %s %v %v", call.GetToolCallId(), call.GetTitle(), call.GetKind(), call.GetStatus(), call.GetContent(), paths)
	case *acp.TaskFrame_ToolCallUpdate:
		change := update.ToolCallUpdate
		return fmt.Sprintf("update %s %s", change.GetToolCallId(), change.GetStatus())
	}
	return "unknown"
}

// recordRun is the part of a script that writes its folder to out.dir, each argument on a line of its
// own to out.argv, the files of its folder to out.files, and the MCP configuration and the config.json
// of its home to out.mcp and out.cfg.
func recordRun(out string) string {
	return "pwd > " + quoteShell(out+".dir") + "\n" +
		"ls -A > " + quoteShell(out+".files") + "\n" +
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + quoteShell(out+".argv") + "\n" +
		"cat \"$HOME/.gemini/config/mcp_config.json\" > " + quoteShell(out+".mcp") + "\n" +
		"cat \"$HOME/.gemini/config/config.json\" > " + quoteShell(out+".cfg") + " 2>/dev/null || true\n"
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func hasReason(err error, want connection.Reason) bool {
	return err != nil && err.Error() == want.String()
}
