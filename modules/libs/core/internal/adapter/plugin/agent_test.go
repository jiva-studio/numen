package plugin

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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
				Length:  10,
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
		Step: &v1.AskAgentResponse_Text{
			Text: "answer: " + req.Msg.GetPrompt(),
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
		"id": "md.numen.agents.numen",
		"name": "Agent Plugin",
		"version": "1.0.0",
		"description": "Agent and chat provider",
		"capabilities": ["ai.agent.provider"],
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
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestAgentBridge(t *testing.T) {
	bridge := openTestBridge(t, &mockAgentServer{})
	ctx := t.Context()

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
			Step: &v1.AskAgentResponse_Text{
				Text: fmt.Sprintf("msg %d", i),
			},
		})
	}
	return nil
}

func TestAgentBridge_UnreadChannelDoesNotDeadlock(t *testing.T) {
	bridge := openTestBridge(t, &floodAgentServer{})
	ctx := t.Context()

	run, err := bridge.StartRun(ctx, port.Task{Question: "test flood"})
	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	stopCh := make(chan struct{})
	go func() {
		_ = run.Stop()
		close(stopCh)
	}()

	select {
	case <-stopCh:
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

// openTestBridge serves the handler as the agent plugin and opens a bridge to it.
func openTestBridge(t *testing.T, handler numenv1connect.AgentServiceHandler) *AgentBridge {
	t.Helper()

	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewAgentServiceHandler(handler))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
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
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	bridge, err := OpenAgentBridgeForPlugin(t.Context(), host, "md.numen.agents.numen")
	if err != nil {
		t.Fatalf("OpenAgentBridgeForPlugin failed: %v", err)
	}
	t.Cleanup(func() { _ = bridge.Close() })
	return bridge
}

type programsServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
	asked []*v1.AgentProgramCommand
}

func (s *programsServer) GetAgentPrograms(
	_ context.Context,
	req *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	s.asked = req.Msg.GetCommands()
	return connect.NewResponse(&v1.GetAgentProgramsResponse{Programs: []*v1.AgentProgram{{
		Name: "one", Title: "One", IsInstalled: true, IsDefault: true, LimitsTools: true,
		Models: []*v1.AgentModel{{
			Name: "m1", Title: "M one", IsDefault: true, DefaultReasoning: "low",
			ReasoningLevels: []*v1.ReasoningLevel{{Name: "low", Title: "Low"}, {Name: "high", Title: "High"}},
		}, {Name: "m2"}},
	}}}), nil
}

// The programs a plugin lists come back under the plugin that listed them, and the command lines
// the installation set go to it.
func TestTheBridgeListsTheProgramsAPluginRuns(t *testing.T) {
	served := &programsServer{}
	bridge := openTestBridge(t, served)

	got, err := bridge.GetPrograms(t.Context(), map[string][]string{"one": {"/bin/one", "--x"}})
	if err != nil {
		t.Fatal(err)
	}

	want := []port.AgentProgram{{
		Provider: "md.numen.agents.numen", Name: "one", Title: "One", IsInstalled: true, IsDefault: true, LimitsTools: true,
		Models: []port.AgentModel{{
			Name: "m1", Title: "M one", IsDefault: true, DefaultReasoning: "low",
			ReasoningLevels: []port.ReasoningLevel{{Name: "low", Title: "Low"}, {Name: "high", Title: "High"}},
		}, {Name: "m2"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if len(served.asked) != 1 || served.asked[0].GetProgram() != "one" ||
		!slices.Equal(served.asked[0].GetCommand(), []string{"/bin/one", "--x"}) {
		t.Errorf("the plugin received %v", served.asked)
	}
}

// A plugin that cannot list its programs is an error.
func TestTheBridgeFailsWhereThePluginListsNoPrograms(t *testing.T) {
	bridge := openTestBridge(t, &floodAgentServer{})

	if _, err := bridge.GetPrograms(t.Context(), nil); err == nil {
		t.Error("a plugin that serves no program list listed some")
	}
}

// A limit on steps that no request can carry is the largest one it can, and a negative one is none.
func TestALimitOnStepsIsCachedWithinWhatTheRequestCarries(t *testing.T) {
	for limit, want := range map[int]int32{
		-5:            0,
		0:             0,
		7:             7,
		math.MaxInt32: math.MaxInt32,
		math.MaxInt:   math.MaxInt32,
	} {
		if got := newAskRequest(port.Task{MaxSteps: limit}).GetMaxSteps(); got != want {
			t.Errorf("%d was sent as %d, want %d", limit, got, want)
		}
	}
}

// Everything the task gives the agent reaches the request, and the effect of each tool goes in the
// word the plugins read it in.
func TestTheRequestCarriesWhatTheTaskProducesTheAgent(t *testing.T) {
	got := newAskRequest(port.Task{
		Question: "q", Focus: "f", Conversation: "c", Agent: "one", Model: "m", Reasoning: "high", Instructions: "i",
		Tools: []string{"note_edit"}, MaxSteps: 7, Root: "/vault", Command: []string{"/bin/one"},
		Options:  map[string]string{"k": "true"},
		Endpoint: port.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "t"},
		ToolDeclarations: []port.ToolDeclaration{
			{Name: "note_edit", Title: "Edit a note", Kind: domain.StepEdit, About: "path", Match: "stood", Text: "becomes"},
			{Name: "note_read", Kind: domain.StepToolCall, Element: "path"},
		},
	})

	if got.GetPrompt() != "q" || got.GetFocus() != "f" || got.GetConversation() != "c" || got.GetAgent() != "one" ||
		got.GetModel() != "m" || got.GetReasoning() != "high" || got.GetInstructions() != "i" || got.GetMaxSteps() != 7 ||
		!slices.Equal(got.GetTools(), []string{"note_edit"}) {
		t.Errorf("the question is %v", got)
	}
	if got.GetRoot() != "/vault" || !slices.Equal(got.GetCommand(), []string{"/bin/one"}) ||
		got.GetOptions()["k"] != "true" || got.GetToolsUrl() != "http://127.0.0.1:1/mcp" || got.GetToolsToken() != "t" {
		t.Errorf("what the agent is given is %v", got)
	}
	declared := got.GetToolDeclarations()
	if len(declared) != 2 {
		t.Fatalf("the declarations are %v", declared)
	}
	if edit := declared[0]; edit.GetName() != "note_edit" || edit.GetTitle() != "Edit a note" ||
		edit.GetKind() != "edit" || edit.GetAbout() != "path" || edit.GetMatch() != "stood" || edit.GetText() != "becomes" {
		t.Errorf("the edit tool is declared as %v", edit)
	}
	if read := declared[1]; read.GetKind() != "" || read.GetElement() != "path" {
		t.Errorf("the other tool is declared as %v", read)
	}
}

// The word a tool's effect is told in is the word it is read back from.
func TestAnEffectIsReportedAndReadInTheSameWord(t *testing.T) {
	for _, effect := range []domain.StepKind{
		domain.StepRead, domain.StepEdit, domain.StepRemove, domain.StepMove, domain.StepSearch, domain.StepToolCall,
	} {
		if got := getStepKind(getKindWord(effect)); got != effect {
			t.Errorf("%v received as %q and read as %v", effect, getKindWord(effect), got)
		}
	}
}

// A change being written reaches the core as a draft of the note, with the text it replaces and
// the text that takes its place so far.
func TestAChangeBeingSavedIsADraft(t *testing.T) {
	got := convertResponseStep(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_EditProgress{
		EditProgress: &v1.EditProgress{Call: "toolu_01", Path: "a/b.md", Match: "old", Text: "new"},
	}})

	want := port.Step{
		Kind: domain.StepDraft, Call: "toolu_01", Place: domain.Place{Path: "a/b.md"}, Match: "old", Text: "new",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

type failingAgentServer struct {
	numenv1connect.UnimplementedAgentServiceHandler
}

func (s *failingAgentServer) AskAgent(
	_ context.Context,
	_ *connect.Request[v1.AskAgentRequest],
	stream *connect.ServerStream[v1.AskAgentResponse],
) error {
	_ = stream.Send(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_Text{Text: "partial"}})
	return connect.NewError(connect.CodeInternal, fmt.Errorf("the program crashed"))
}

// A stream the plugin ends with an error is a run that stopped, and the error is why.
func TestARunWhoseStreamFailsStopsWithTheError(t *testing.T) {
	bridge := openTestBridge(t, &failingAgentServer{})

	run, err := bridge.StartRun(t.Context(), port.Task{Question: "q"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = run.Stop() }()

	var steps []port.Step
	for step := range run.GetSteps() {
		steps = append(steps, step)
	}
	if len(steps) != 2 || steps[0].Text != "partial" || steps[1].Kind != domain.StepStop ||
		!strings.Contains(steps[1].Detail, "the program crashed") {
		t.Errorf("the steps are %+v", steps)
	}
}
