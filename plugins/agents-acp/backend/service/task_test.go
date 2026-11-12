package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools/toolstest"
)

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

func taskCall(connectionType, program string) connection.Call {
	return connection.Call{ConnectionID: "conn-orchard", Type: connectionType, Fields: map[string]string{"program": program}}
}

func runTask(ctx context.Context, svc *agentServer, call connection.Call, request *acp.TaskRequest) ([]string, *acp.TaskDone, error) {
	var frames []string
	done, err := svc.tasks.RunTask(ctx, call, request, func(frame *acp.TaskFrame) error {
		frames = append(frames, describeFrame(frame))
		return nil
	})
	return frames, done, err
}

func describeFrame(frame *acp.TaskFrame) string {
	switch update := frame.GetUpdate().(type) {
	case *acp.TaskFrame_AgentMessageChunk:
		return fmt.Sprintf("chunk %s %q", update.AgentMessageChunk.GetMessageId(), update.AgentMessageChunk.GetText())
	case *acp.TaskFrame_Plan:
		var entries []string
		for _, one := range update.Plan.GetEntries() {
			entries = append(entries, one.GetContent()+" "+one.GetPriority().String()+" "+one.GetStatus().String())
		}
		return "plan " + strings.Join(entries, ", ")
	case *acp.TaskFrame_ToolCall:
		call := update.ToolCall
		var paths []string
		for _, one := range call.GetLocations() {
			paths = append(paths, one.GetPath())
		}
		return fmt.Sprintf("call %s %q %s %s %v %v", call.GetToolCallId(), call.GetTitle(), call.GetKind(), call.GetStatus(), call.GetContent(), paths)
	case *acp.TaskFrame_ToolCallUpdate:
		change := update.ToolCallUpdate
		return fmt.Sprintf("update %s %s %s", change.GetToolCallId(), change.GetStatus(), change.GetRawOutputJson())
	}
	return "unknown"
}

func getPromptTexts(t *testing.T, prompt map[string]any) []string {
	t.Helper()
	var texts []string
	for _, block := range getField[[]any](t, getField[map[string]any](t, prompt, "params"), "prompt") {
		texts = append(texts, getField[string](t, block, "text"))
	}
	return texts
}

func TestATaskStreamsTheAnswerAsItArrivesAndEndsWithTheStopReason(t *testing.T) {
	program, _ := fakeProgram(t, "stream")

	frames, done, err := runTask(t.Context(), newTestServer(t), taskCall("acp", program), &acp.TaskRequest{Prompt: "How are the beds?"})

	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN {
		t.Fatalf("RunTask = %v, %v, want the end of the turn", done, err)
	}
	want := []string{
		`chunk m1 "Raised "`,
		"plan Read the garden note PLAN_ENTRY_PRIORITY_HIGH PLAN_ENTRY_STATUS_PENDING",
		`chunk m2 "beds."`,
	}
	if !slices.Equal(frames, want) {
		t.Errorf("frames = %q, want %q", frames, want)
	}
}

// The program reaches the tools of the host on the MCP server of its session, behind the token of the
// session, and every call runs on the host with the tool context and the vault of the task.
func TestTheToolsOfTheHostAreServedToTheProgramAndCalledWithTheContextAndTheVault(t *testing.T) {
	host, client := serveGardenHost(t)
	program, record := fakeProgram(t, "mcp")

	frames, done, err := runTask(t.Context(), newTestServerWith(t, client, clock.System{}), taskCall("claude-code", program), &acp.TaskRequest{
		Prompt: "How are the beds?", ToolContext: map[string]string{"org.example.garden/bed": "north"}, VaultId: "vault-orchard",
	})

	if err != nil || done.GetStopReason() != acp.StopReason_STOP_REASON_END_TURN {
		t.Fatalf("RunTask = %v, %v", done, err)
	}
	observed := readRecord(t, record)
	if listed := observed["mcp/tools"]; len(listed) != 1 || fmt.Sprint(listed[0]["names"]) != "[garden_plan note_read]" {
		t.Errorf("the program was served %v, want the tools of the host", listed)
	}
	calls := host.listCalls()
	if len(calls) != 1 || calls[0].Name != "note_read" || string(calls[0].Arguments) != `{"path":"Garden.md"}` ||
		!maps.Equal(calls[0].Context, map[string]string{"org.example.garden/bed": "north"}) || calls[0].VaultID != "vault-orchard" {
		t.Errorf("the host ran %+v, want note_read of Garden.md with the context and the vault of the task", calls)
	}
	want := []string{
		`call c1 "Read a note" TOOL_KIND_READ TOOL_CALL_STATUS_PENDING [Garden.md] [Garden.md]`,
		`update c1 TOOL_CALL_STATUS_COMPLETED {"text":"# Garden\nRaised beds."}`,
		`chunk m1 "# Garden\nRaised beds."`,
	}
	if !slices.Equal(frames, want) {
		t.Errorf("frames = %q, want %q", frames, want)
	}
	if prompts := observed["session/prompt"]; len(prompts) != 1 || getPromptTexts(t, prompts[0])[0] != "Cite the note you read. Ask before you plan." {
		t.Errorf("the prompts were %v, want the instructions of the tools first", prompts)
	}
}

// Claude Code is started with its own tools off but web search, the served tools allowed, and no settings
// read; the session is offered the endpoint of the tools as its one MCP server.
func TestClaudeCodeOpensASessionWithTheServedToolsAlone(t *testing.T) {
	_, client := serveGardenHost(t)
	program, record := fakeProgram(t, "mcp")

	if _, _, err := runTask(t.Context(), newTestServerWith(t, client, clock.System{}), taskCall("claude-code", program), &acp.TaskRequest{Prompt: "Plan the beds."}); err != nil {
		t.Fatal(err)
	}

	params := getField[map[string]any](t, readRecord(t, record)["session/new"][0], "params")
	meta, err := json.Marshal(params["_meta"])
	if err != nil {
		t.Fatal(err)
	}
	wantMeta := `{"claudeCode":{"options":{"allowedTools":["WebSearch","mcp__numen__garden_plan","mcp__numen__note_read"],` +
		`"maxTurns":100,"settingSources":[],"tools":["WebSearch"]}}}`
	if string(meta) != wantMeta {
		t.Errorf("the session was opened with %s, want %s", meta, wantMeta)
	}
	servers := getField[[]any](t, params, "mcpServers")
	if len(servers) != 1 || getField[string](t, servers[0], "name") != "numen" || getField[string](t, servers[0], "type") != "http" ||
		!strings.HasPrefix(getField[string](t, servers[0], "url"), "http://127.0.0.1:") {
		t.Errorf("the session was offered %v, want the endpoint of the tools on loopback", servers)
	}
}

// A permission is granted once to a call of a served tool, named by the call, by its Codex wrapping or
// by what the agent reported for the call earlier; a title naming a tool grants nothing. The rest is
// rejected with the option that rejects once.
func TestOnlyTheServedToolsAreAllowed(t *testing.T) {
	_, client := serveGardenHost(t)
	program, record := fakeProgram(t, "tools")

	frames, _, err := runTask(t.Context(), newTestServerWith(t, client, clock.System{}), taskCall("acp", program), &acp.TaskRequest{Prompt: "Read the garden."})
	if err != nil {
		t.Fatal(err)
	}

	replies := readRecord(t, record)["reply"]
	if len(replies) < 6 {
		t.Fatalf("the agent had %d replies", len(replies))
	}
	allowed, rejected := `{"outcome":{"optionId":"once","outcome":"selected"}}`, `{"outcome":{"optionId":"no","outcome":"selected"}}`
	for i, want := range []string{allowed, rejected, rejected, allowed, allowed, rejected} {
		outcome, err := json.Marshal(replies[i]["result"])
		if err != nil {
			t.Fatal(err)
		}
		if string(outcome) != want {
			t.Errorf("permission %d was answered %s, want %s", i, outcome, want)
		}
	}
	wantFrames := []string{
		`call c0 "Read a note" TOOL_KIND_READ TOOL_CALL_STATUS_PENDING [Garden.md] [Garden.md]`,
		`update c0 TOOL_CALL_STATUS_IN_PROGRESS `,
		`update c0 TOOL_CALL_STATUS_COMPLETED `,
	}
	if len(frames) < 3 || !slices.Equal(frames[:3], wantFrames) {
		t.Errorf("frames = %q, want the call once and its end once first: %q", frames, wantFrames)
	}
}

func TestATurnOfAConversationContinuesItsSession(t *testing.T) {
	_, client := serveGardenHost(t)
	program, record := fakeProgram(t, "plain")
	svc := newTestServerWith(t, client, clock.System{})

	for _, prompt := range []string{"Which beds face north?", "And south?"} {
		if _, _, err := runTask(t.Context(), svc, taskCall("acp", program), &acp.TaskRequest{Prompt: prompt, Conversation: "talk-garden"}); err != nil {
			t.Fatal(err)
		}
	}

	observed := readRecord(t, record)
	if opened, prompted := len(observed["session/new"]), len(observed["session/prompt"]); opened != 1 || prompted != 2 {
		t.Fatalf("the program opened %d sessions for %d prompts, want one session for two", opened, prompted)
	}
	if texts := getPromptTexts(t, observed["session/prompt"][1]); !slices.Equal(texts, []string{"And south?"}) {
		t.Errorf("the second turn sent %q, want the prompt alone", texts)
	}
	if closed := len(observed["session/close"]); closed != 0 {
		t.Errorf("the session of the conversation was closed %d times", closed)
	}
}

func TestATaskWithNoConversationHasASessionOfItsOwnClosedAtTheEnd(t *testing.T) {
	program, record := fakeProgram(t, "plain")
	svc := newTestServer(t)

	for range 2 {
		if _, _, err := runTask(t.Context(), svc, taskCall("acp", program), &acp.TaskRequest{Prompt: "Which beds face north?"}); err != nil {
			t.Fatal(err)
		}
	}

	observed := readRecord(t, record)
	if opened, closed := len(observed["session/new"]), len(observed["session/close"]); opened != 2 || closed != 2 {
		t.Errorf("the program opened %d sessions and closed %d, want two of each", opened, closed)
	}
}

func TestASessionIdleForThirtyMinutesIsClosed(t *testing.T) {
	program, record := fakeProgram(t, "plain")
	source := newFakeClock(time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC))
	svc := newTestServerWith(t, nil, source)
	request := &acp.TaskRequest{Prompt: "Which beds face north?", Conversation: "talk-garden"}

	if _, _, err := runTask(t.Context(), svc, taskCall("acp", program), request); err != nil {
		t.Fatal(err)
	}
	source.Advance(29 * time.Minute)
	if err := svc.tasks.CloseIdle(t.Context()); err != nil || len(readRecord(t, record)["session/close"]) != 0 {
		t.Fatalf("a session idle for 29 minutes was closed (%v)", err)
	}
	source.Advance(time.Minute)
	if err := svc.tasks.CloseIdle(t.Context()); err != nil {
		t.Fatal(err)
	}
	if closed := len(readRecord(t, record)["session/close"]); closed != 1 {
		t.Fatalf("a session idle for 30 minutes was closed %d times, want once", closed)
	}
	if _, _, err := runTask(t.Context(), svc, taskCall("acp", program), request); err != nil {
		t.Fatal(err)
	}
	if opened := len(readRecord(t, record)["session/new"]); opened != 2 {
		t.Errorf("the program opened %d sessions, want a new one after the idle close", opened)
	}
}

func TestThePluginKeepsTheSessionsOfEightConversations(t *testing.T) {
	program, record := fakeProgram(t, "plain")
	source := newFakeClock(time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC))
	svc := newTestServerWith(t, nil, source)

	for i := range 9 {
		source.Advance(time.Second)
		request := &acp.TaskRequest{Prompt: "Which beds face north?", Conversation: fmt.Sprintf("talk-%d", i)}
		if _, _, err := runTask(t.Context(), svc, taskCall("acp", program), request); err != nil {
			t.Fatal(err)
		}
	}
	if closed := len(readRecord(t, record)["session/close"]); closed != 1 {
		t.Fatalf("%d sessions were closed for the ninth conversation, want the least recently used one", closed)
	}
	if err := svc.tasks.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if closed := len(readRecord(t, record)["session/close"]); closed != 9 {
		t.Errorf("%d sessions were closed when the plugin stopped, want all nine", closed)
	}
}

func TestACanceledTaskCancelsTheTurnInTheProgram(t *testing.T) {
	program, record := fakeProgram(t, "hang")
	ctx, cancel := context.WithCancel(t.Context())
	go cancelOnPrompt(record, cancel)

	_, _, err := runTask(ctx, newTestServer(t), taskCall("acp", program), &acp.TaskRequest{Prompt: "Which beds face north?", Conversation: "talk-garden"})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunTask = %v, want the cancellation", err)
	}
	if cancels := len(readRecord(t, record)["session/cancel"]); cancels != 1 {
		t.Errorf("the program was sent %d cancels, want one", cancels)
	}
}

func TestAProgramThatIsMissingOrFailsIsTriedNext(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	failing, _ := fakeProgram(t, "fail")
	noHTTP, _ := fakeProgram(t, "no-http")
	_, client := serveGardenHost(t)
	svc := newTestServerWith(t, client, clock.System{})

	for name, call := range map[string]connection.Call{
		"a missing program":          taskCall("acp", filepath.Join(t.TempDir(), "gone")),
		"a known binary not on PATH": taskCall("codex", ""),
		"a program that exits":       taskCall("acp", failing),
		"a program without HTTP MCP": taskCall("acp", noHTTP),
	} {
		_, _, err := runTask(t.Context(), svc, call, &acp.TaskRequest{Prompt: "Which beds face north?"})
		if _, isTryNext := connection.IsTryNext(err); !isTryNext {
			t.Errorf("%s: RunTask = %v, want try the next", name, err)
		}
	}
}

func TestAnACPConnectionWithoutAProgramIsInvalid(t *testing.T) {
	_, _, err := runTask(t.Context(), newTestServer(t), taskCall("acp", ""), &acp.TaskRequest{Prompt: "Which beds face north?"})

	if !connection.IsStop(err) || err.Error() != connection.ReasonInvalidRequest.String() {
		t.Errorf("RunTask = %v, want a stop for an invalid request", err)
	}
}

func TestAFrameThatCannotBeSentEndsTheTaskWithItsError(t *testing.T) {
	program, _ := fakeProgram(t, "stream")
	errSend := errors.New("the window is gone")

	_, err := newTestServer(t).tasks.RunTask(t.Context(), taskCall("acp", program), &acp.TaskRequest{Prompt: "How are the beds?"},
		func(*acp.TaskFrame) error { return errSend })

	if !errors.Is(err, errSend) {
		t.Errorf("RunTask = %v, want the error of the send", err)
	}
}

func cancelOnPrompt(record string, cancel context.CancelFunc) {
	defer cancel()
	deadline := time.After(5 * time.Second)
	for {
		if raw, err := os.ReadFile(record); err == nil && bytes.Contains(raw, []byte(`"method":"session/prompt"`)) {
			return
		}
		select {
		case <-deadline:
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// newFakeClock is a clock that reports start until the test moves it.
func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}
