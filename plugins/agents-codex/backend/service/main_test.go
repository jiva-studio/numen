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
	"strconv"
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
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/task"
)

// TestMain runs the test binary as a fake Codex when it is started as one, and the tests of m otherwise.
func TestMain(m *testing.M) {
	if len(os.Args) > 2 && os.Args[1] == "fake-codex" {
		time.AfterFunc(20*time.Second, func() { os.Exit(3) })
		os.Exit(runFakeCodex(os.Args[3:], os.Stdout))
	}
	os.Exit(m.Run())
}

func runFakeCodex(args []string, out io.Writer) int {
	url := ""
	for _, one := range args {
		if value, isURL := strings.CutPrefix(one, "mcp_servers.numen.url="); isURL {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return 1
			}
			url = unquoted
		}
	}
	var lines []string
	if url != "" {
		answer, err := callNoteRead(url, os.Getenv("NUMEN_TOKEN"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 2
		}
		lines = append(lines,
			`{"type":"item.started","item":{"id":"i1","type":"mcp_tool_call","server":"numen","tool":"note_read","arguments":{"path":"Garden/Beds.md"}}}`,
			`{"type":"item.completed","item":{"id":"i1","type":"mcp_tool_call","status":"completed","result":`+quoteJSON(answer)+`}}`,
		)
	}
	lines = append(lines, `{"type":"item.completed","item":{"id":"i2","type":"agent_message","text":"Raised beds."}}`, `{"type":"turn.completed"}`)
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

func callNoteRead(url, token string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
		req.Header.Set("Authorization", "Bearer "+token)
		return http.DefaultTransport.RoundTrip(req)
	})}
	session, err := sdk.NewClient(&sdk.Implementation{Name: "fake-codex", Version: "1"}, nil).
		Connect(ctx, &sdk.StreamableClientTransport{Endpoint: url, HTTPClient: client, DisableStandaloneSSE: true}, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := session.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "close the session:", err)
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

func fakeCodex(t *testing.T) string {
	t.Helper()
	return writeScript(t, "codex", "exec "+quoteShell(os.Args[0])+" fake-codex run \"$@\"\n")
}

func quoteShell(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	needsShell(t)
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}

func needsShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
}

func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if found := codexcli.FindCodex(); found != "codex" {
		t.Skipf("Codex is installed at %s", found)
	}
}

func recordArgs(dir string) string {
	return "pwd > " + dir + "/pwd\n" +
		"for a in \"$@\"; do printf '%s\\0' \"$a\"; done > " + dir + "/argv\n" +
		"cat > " + dir + "/stdin\n" +
		"printf '%s' \"$NUMEN_TOKEN\" > " + dir + "/token\n"
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func readArgv(t *testing.T, dir string) []string {
	t.Helper()
	return strings.Split(strings.TrimSuffix(readFile(t, filepath.Join(dir, "argv")), "\x00"), "\x00")
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

func newTestServer(t *testing.T, host task.Tools) *agentServer {
	t.Helper()
	built, err := newAgentServer(parts{tools: host, isAsync: false})
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

func newCodex(t *testing.T, host task.Tools, program string) *agentServer {
	t.Helper()
	built := newTestServer(t, host)
	built.settings.Apply(map[string]string{"executablePath": program})
	return built
}

func taskCall(options map[string]string) connection.Call {
	return connection.Call{ConnectionID: "conn-orchard", Type: "codex", Options: options}
}

func runTask(ctx context.Context, svc *agentServer, call connection.Call, request *acp.TaskRequest) ([]string, *acp.TaskDone, error) {
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
	case *acp.TaskFrame_AgentThoughtChunk:
		return fmt.Sprintf("thought %s %q", update.AgentThoughtChunk.GetMessageId(), update.AgentThoughtChunk.GetText())
	}
	return "unknown"
}

func hasReason(err error, want connection.Reason) bool {
	return err != nil && err.Error() == want.String()
}
