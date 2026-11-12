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
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/internal/adapter/hostenv"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/usecase/task"
)

// TestMain runs the test binary as a fake Claude Code when it is started as one, and the tests of m
// otherwise.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "fake-claude" {
		time.AfterFunc(20*time.Second, func() { os.Exit(3) })
		os.Exit(runFakeClaude(os.Args[2], os.Args[3:], os.Stdin, os.Stdout))
	}
	os.Exit(m.Run())
}

func runFakeClaude(record string, args []string, in io.Reader, out io.Writer) int {
	prompt, err := io.ReadAll(in)
	if err != nil {
		return 1
	}
	config := ""
	for place, one := range args {
		if one == "--mcp-config" && place+1 < len(args) {
			config = args[place+1]
		}
	}
	raw, err := json.Marshal(map[string]any{"args": args, "prompt": string(prompt)})
	if err != nil || os.WriteFile(record, raw, 0o600) != nil {
		return 1
	}
	lines := []string{`{"type":"system","subtype":"init","session_id":"s-garden","mcp_servers":[{"name":"numen","status":"connected"}]}`}
	if config != "" {
		answer, err := callNoteRead(config)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 2
		}
		lines = append(lines,
			`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"c1","name":"mcp__numen__note_read","input":{"path":"Garden/Beds.md"}}]}}`,
			`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"c1","content":`+quoteJSON(answer)+`}]}}`,
		)
	}
	lines = append(lines,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Raised "}}}`,
		`{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"beds."}}}`,
		`{"type":"result","is_error":false,"result":"Raised beds."}`,
	)
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
			URL     string            `json:"url"`
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
	session, err := sdk.NewClient(&sdk.Implementation{Name: "fake-claude", Version: "1"}, nil).
		Connect(ctx, &sdk.StreamableClientTransport{Endpoint: server.URL, HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := session.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
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

func fakeClaude(t *testing.T) (program, record string) {
	t.Helper()
	record = filepath.Join(t.TempDir(), "record")
	return writeScript(t, "claude", "exec "+quoteShell(os.Args[0])+" fake-claude "+quoteShell(record)+` "$@"`+"\n"), record
}

type fakeRun struct {
	Args   []string `json:"args"`
	Prompt string   `json:"prompt"`
}

func readRun(t *testing.T, record string) fakeRun {
	t.Helper()
	raw, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	var run fakeRun
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	return run
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

func withoutClaudeAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
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
	built, err := newAgent(parts{environment: hostenv.Process{}, tools: host, isAsync: false})
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

func newClaude(t *testing.T, host task.Tools, program string) *agent {
	t.Helper()
	built := newTestAgent(t, host)
	built.settings.Apply(map[string]string{"executablePath": program})
	return built
}

func taskCall(options map[string]string) connection.Call {
	return connection.Call{ConnectionID: "conn-orchard", Type: "claude-code", Options: options}
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

func readArgv(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
}

func assertPair(t *testing.T, argv []string, flag, value string) {
	t.Helper()
	for place, one := range argv {
		if one == flag && place+1 < len(argv) {
			if argv[place+1] != value {
				t.Errorf("%s %q, want %q", flag, argv[place+1], value)
			}
			return
		}
	}
	t.Errorf("%s is not among %q", flag, argv)
}

func recordArgs(dir string) string {
	return "cat > " + dir + "/stdin\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + dir + "/argv\n"
}

func hasReason(err error, want connection.Reason) bool {
	return err != nil && err.Error() == want.String()
}
