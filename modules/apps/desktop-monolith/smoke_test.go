package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

const (
	agentID    = "md.numen.agent"
	readerID   = "md.numen.pdf"
	modelReply = "The vault holds three notes."
)

// answers is what one build gave to the three calls of the smoke test.
type answers struct {
	Chat     string
	Single   string
	Pages    int
	Programs []string
	// AgentPrograms is the names of the programs each coding agent plugin lists.
	AgentPrograms map[string][]string
}

// newChatAPI is an OpenAI-compatible chat API that answers every question with modelReply, as one
// message or, where the request asks for a stream, as the chunks of one.
func newChatAPI(t *testing.T) *httptest.Server {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": "small-model", "name": "Small"}}})
			return
		}
		var request struct {
			Stream bool `json:"stream"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if !request.Stream {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": modelReply}, "finish_reason": "stop"},
			}})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range []map[string]any{
			{"choices": []map[string]any{{"delta": map[string]string{"role": "assistant", "content": modelReply}}}},
			{"choices": []map[string]any{{"delta": map[string]string{}, "finish_reason": "stop"}}},
		} {
			raw, _ := json.Marshal(chunk)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(api.Close)
	return api
}

// buildPlugin compiles a plugin's backend into a folder the host discovers it from, with its own
// manifest.json beside it.
func buildPlugin(t *testing.T, folder, name string) {
	t.Helper()
	plugin := filepath.Join("..", "..", "plugins", name)
	manifest, err := os.ReadFile(filepath.Join(plugin, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var declared struct {
		ID      string `json:"id"`
		Backend struct {
			Entrypoint string `json:"entrypoint"`
		} `json:"backend"`
	}
	if err := json.Unmarshal(manifest, &declared); err != nil {
		t.Fatal(err)
	}
	home, err := filepath.Abs(filepath.Join(folder, declared.ID))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, filepath.Dir(declared.Backend.Entrypoint)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(home, declared.Backend.Entrypoint), "./backend")
	build.Dir = plugin
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", name, err, out)
	}
}

// newLaunchedHost is a plugin host that launches the plugins as processes, as the modular build does.
func newLaunchedHost(t *testing.T) pluginport.PluginManager {
	t.Helper()
	// The host does not run a binary from the temporary directory, so the plugins are built beside the test.
	folder, err := os.MkdirTemp(".", "plugins-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(folder) })
	buildPlugin(t, folder, "agent")
	buildPlugin(t, folder, "reader-pdf")
	for _, id := range codingAgentIDs {
		buildPlugin(t, folder, strings.TrimPrefix(id, "md.numen."))
	}
	host := container.NewPluginHost("desktop.editor", folder)
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), 2*time.Second) })
	return host
}

// askBuild puts a chat, a single answer and a document to the plugins of a host.
func askBuild(t *testing.T, host pluginport.PluginManager, chatURL string) answers {
	t.Helper()
	host.SetKeyStore(heldKeys{"md.numen.agent/apiKey": "chat-key"})
	host.SetSettings(map[string]map[string]any{
		agentID: {"baseUrl": chatURL, "model": "small-model", pluginport.SettingStoredSecrets: []any{"apiKey"}},
	})

	agent, err := host.EnsureActive(t.Context(), agentID)
	if err != nil {
		t.Fatalf("activate the agent: %v", err)
	}
	agentClient := newClient(agent)
	var got answers

	agentService := numenv1connect.NewAgentServiceClient(agentClient, agent.BaseURL(), withToken(agent.Token))
	chat := ask(t, agentService, &numenv1.AskAgentRequest{Prompt: "How many notes are there?"})
	got.Chat = chat
	got.Single = ask(t, agentService, &numenv1.AskAgentRequest{Prompt: "How many notes are there?"})

	programs, err := numenv1connect.NewAgentServiceClient(agentClient, agent.BaseURL(), withToken(agent.Token)).
		GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatalf("GetAgentPrograms: %v", err)
	}
	for _, program := range programs.Msg.GetPrograms() {
		got.Programs = append(got.Programs, program.GetName())
	}

	got.AgentPrograms = listCodingAgentPrograms(t, host)

	reader, err := host.EnsureActive(t.Context(), readerID)
	if err != nil {
		t.Fatalf("activate the reader: %v", err)
	}
	path, err := filepath.Abs(filepath.Join("..", "..", "plugins", "reader-pdf", "backend", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := numenv1connect.NewDocumentServiceClient(newClient(reader), reader.BaseURL(), withToken(reader.Token)).
		GetDocument(t.Context(), connect.NewRequest(&numenv1.GetDocumentRequest{Path: path}))
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}
	got.Pages = len(document.Msg.GetPages())
	return got
}

// ask puts the task to the agent and returns what it said.
func ask(t *testing.T, client numenv1connect.AgentServiceClient, task *numenv1.AskAgentRequest) string {
	t.Helper()
	stream, err := client.AskAgent(t.Context(), connect.NewRequest(task))
	if err != nil {
		t.Fatalf("AskAgent: %v", err)
	}
	var response strings.Builder
	for stream.Receive() {
		response.WriteString(stream.Msg().GetText())
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("AskAgent stream: %v", err)
	}
	return response.String()
}

// newClient is the HTTP client that reaches an endpoint, in-process or on a port.
func newClient(endpoint pluginport.Endpoint) *http.Client {
	if transport, ok := endpoint.Transport.(http.RoundTripper); ok {
		return &http.Client{Transport: transport}
	}
	return &http.Client{}
}

// withToken presents the token of an endpoint on every call to it.
func withToken(token string) connect.ClientOption {
	return connect.WithInterceptors(&bearerToken{token: token})
}

type bearerToken struct{ token string }

func (b *bearerToken) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("Authorization", "Bearer "+b.token)
		return next(ctx, req)
	}
}

func (b *bearerToken) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("Authorization", "Bearer "+b.token)
		return conn
	}
}

func (b *bearerToken) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// The build that links the plugins into its process and the build that launches them answer a chat,
// a single answer and a document alike, and declare the same capabilities.
func TestBothBuildsAnswerAlike(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the plugins as processes")
	}
	chat := newChatAPI(t)
	linked, launched := newLinkedConfig(t).PluginHost, newLaunchedHost(t)

	fromLinked := askBuild(t, linked, chat.URL)
	fromLaunched := askBuild(t, launched, chat.URL)

	if fromLinked.Chat != modelReply || fromLinked.Single != modelReply || fromLinked.Pages == 0 {
		t.Errorf("the linked build answered %+v", fromLinked)
	}
	if fromLaunched.Chat != fromLinked.Chat || fromLaunched.Single != fromLinked.Single ||
		fromLaunched.Pages != fromLinked.Pages || !slices.Equal(fromLaunched.Programs, fromLinked.Programs) {
		t.Errorf("the launched build answered %+v, the linked build %+v", fromLaunched, fromLinked)
	}
	for id, want := range map[string][]string{"md.numen.agent-acp": nil, "md.numen.agent-codex": {"codex"}} {
		if got := fromLinked.AgentPrograms[id]; !slices.Equal(got, want) {
			t.Errorf("%s lists %v, want %v", id, got, want)
		}
	}
	if !maps.EqualFunc(fromLaunched.AgentPrograms, fromLinked.AgentPrograms, slices.Equal) {
		t.Errorf("the launched build lists %v, the linked build %v", fromLaunched.AgentPrograms, fromLinked.AgentPrograms)
	}
	for _, id := range []string{agentID, readerID} {
		inProcess, ok1 := linked.GetPlugin(id)
		onDisk, ok2 := launched.GetPlugin(id)
		if !ok1 || !ok2 || !slices.Equal(inProcess.Manifest.Capabilities, onDisk.Manifest.Capabilities) {
			t.Errorf("%s declares %v linked and %v launched", id, inProcess.Manifest.Capabilities, onDisk.Manifest.Capabilities)
		}
	}
}

// codingAgentIDs are the plugins that run a coding agent program. The folder of each is its id without
// the md.numen. prefix.
var codingAgentIDs = []string{"md.numen.agent-acp", "md.numen.agent-antigravity", "md.numen.agent-claude", "md.numen.agent-codex"}

// listCodingAgentPrograms asks each coding agent plugin of a host for the names of its programs. A
// program is listed whether or not its command is installed on the machine.
func listCodingAgentPrograms(t *testing.T, host pluginport.PluginManager) map[string][]string {
	t.Helper()
	// Codex is asked for its models through a command that does not exist, so no installed CLI is started.
	absentCodex := []*numenv1.AgentProgramCommand{{Program: "codex", Command: []string{filepath.Join(t.TempDir(), "absent")}}}
	items := make(map[string][]string, len(codingAgentIDs))
	for _, id := range codingAgentIDs {
		endpoint, err := host.EnsureActive(t.Context(), id)
		if err != nil {
			t.Fatalf("activate %s: %v", id, err)
		}
		res, err := numenv1connect.NewAgentServiceClient(newClient(endpoint), endpoint.BaseURL(), withToken(endpoint.Token)).
			GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{Commands: absentCodex}))
		if err != nil {
			t.Fatalf("GetAgentPrograms of %s: %v", id, err)
		}
		items[id] = nil
		for _, program := range res.Msg.GetPrograms() {
			items[id] = append(items[id], program.GetName())
		}
	}
	return items
}
